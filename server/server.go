package server

import (
	"encoding/json"
	"fmt"
	"logauditer/api"
	"logauditer/command"
	"sort"
	"strings"

	"logauditer/dbapi"
	ii "logauditer/internal"
	ll "logauditer/logmining"

	"net"
	"sync"
	"unsafe"

	"github.com/gogo/grpc-example/insecure"
	"github.com/grpc-ecosystem/go-grpc-middleware/validator"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/mgo.v2/bson"

	log "github.com/laik/logger"
)

const (
	AUDIT_LOG_DATABASE = "audit_rule"
	AUDIT_LOG_RULE     = "data"
)

// 后台调度器
type Scheduler struct {
	mu      sync.Mutex
	workers map[string]*Worker
}

func (s *Scheduler) List() []string {
	result := make([]string, 0, len(s.workers))
	for name, _ := range s.workers {
		result = append(result, name)
	}
	return result
}

func (s *Scheduler) Exists(w *Worker) bool {
	if _, ok := s.workers[w.name]; ok {
		return true
	}
	return false
}

func (s *Scheduler) Add(w *Worker) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := w.run(); err != nil {
		log.Error("run worker error %s\n", err)
		return false
	}
	s.workers[w.name] = w
	return true
}

func (s *Scheduler) Del(w *Worker) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _w, ok := s.workers[w.name]; ok {
		if err := _w.stop(); err != nil {
			return false
		}
	}
	delete(s.workers, w.name)
	return true
}

type Worker struct {
	name string
	d    *ll.Directory
	// runtime options stge
	stge command.DataStore

	persists *dbapi.StorageParts
}

func (w *Worker) run() error {
	ss, err := w.stge.Get(w.name)
	if err != nil {
		return err
	}
	rops := &ii.RuntimeOptions{}

	if err := rops.Unmarshal([]byte(ss), json.Unmarshal); err != nil {
		return err
	}
	log.Debug("runtimeops = %#v\n", rops)
	w.d, err = ll.NewDirectory(rops, ll.ROOT, w.persists, w.name)

	if err != nil {
		return err
	}
	log.Info("start worker (%s)\n", w.name)
	return nil
}

func (w *Worker) stop() (err error) {
	defer log.Info("stop worker (%s) error:(%v)\n", w.name, err)
	w.d.Close()
	return
}

func (w *Worker) list() []string {
	r := make([]string, 0)
	w.d.List(&r)
	sort.Slice(r, func(i, j int) bool { return r[i] < r[j] })
	return r
}

type Server struct {
	logParts    *ii.LogParts
	parser      *command.Parser
	persists    *dbapi.StorageParts
	persistType dbapi.DBType
	scheduler   *Scheduler
	stge        command.DataStore
}

func NewServer(parser *command.Parser, stge command.DataStore, persists *dbapi.StorageParts, persistType dbapi.DBType) (*Server, error) {
	logParts := ii.NewLogParts()
	server := &Server{
		logParts:    logParts,
		parser:      parser,
		persists:    persists,
		persistType: persistType,
		scheduler: &Scheduler{
			workers: make(map[string]*Worker),
		},
		stge: stge,
	}

	var _err error
	var _list []*command.Persist
	dbapi.AccessDatabase(
		server.persists,
		AUDIT_LOG_DATABASE,
		AUDIT_LOG_RULE,
		nil,
		&_list,
		dbapi.KEYS,
		server.persistType,
		&_err,
	)
	if _err != nil {
		log.Error("get persists (%s) not found.\n.", ll.LIBDB)
		return nil, _err
	}
	for _, c := range _list {
		err := stge.Set(c.Id, c.Value)
		if c.Isopen {
			w := &Worker{
				name:     c.Id,
				stge:     server.stge,
				persists: server.persists,
			}
			if !server.scheduler.Exists(w) {
				server.scheduler.Add(w)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("%s", "from persist obtain rule set cache error maybe cache is not initialize.")
		}

	}
	return server, nil
}

func (s *Server) Execute(ctx context.Context, req *api.ExecuteRequest) (*api.ExecuteCommandResponse, error) {
	cmdStr := *(*string)(unsafe.Pointer(&req.Command))

	cmd, args, err := s.parser.Parse(cmdStr)
	if err != nil {
		if err == command.ErrCommandNotFound {
			return &api.ExecuteCommandResponse{
				Reply: api.ErrCommandReply,
				Item:  fmt.Sprintf("command %q not found", req.Command),
			}, err
		}
		return nil, fmt.Errorf("could not parse command: %v", err)
	}

	return s.createExecuteCommandResponse(cmd.Execute(args...))
}

func (s *Server) createExecuteCommandResponse(reply command.Reply) (
	*api.ExecuteCommandResponse,
	error,
) {
	res := new(api.ExecuteCommandResponse)

	switch t := reply.(type) {
	case *command.NilReply:
		res.Reply = api.NilCommandReply

	case *command.OkReply:
		res.Reply = api.OkCommandReply

	case *command.StringReply:
		res.Reply = api.StringCommandReply
		res.Item = t.Message

	case *command.SliceReply:
		res.Reply = api.SliceCommandReply
		if len(t.Message) != 0 {
			res.Items = t.Message
			break
		}
		res.Items = []string{"(noitems)"}

	case *command.PersistReply:
		var _err error
		dbapi.AccessDatabase(
			s.persists,
			AUDIT_LOG_DATABASE,
			AUDIT_LOG_RULE,
			bson.M{"_id": &t.Message.Id},
			&t.Message,
			dbapi.SET,
			s.persistType,
			&_err,
		)
		if _err == nil {
			res.Reply = api.OkCommandReply
			log.Debug("commit rule (%s) success.\n", t.Message.Id)
			break
		}
		res.Reply = api.ErrCommandReply
		res.Item = _err.Error()

	case *command.TestReply:
		tester := t.Message

		runtimeOptions := &ii.RuntimeOptions{
			ColumnPattern: &ii.ColumnPattern{},
		}

		if err := runtimeOptions.Unmarshal(
			[]byte(tester.Rule),
			json.Unmarshal,
		); err != nil {
			res.Reply = api.ErrCommandReply
			res.Item = fmt.Sprintf(
				"test rule error unmarshal to runtimeOptions error: %s.",
				err,
			)
			return res, nil
		}
		var _err error
		resp := ii.NewResponse()
		ii.VisitLogsAudit2(s.logParts, tester.Data, resp, runtimeOptions, &_err)

		if resp.Err != nil || len(strings.Trim(resp.Data, " ")) == 0 {
			res.Item = fmt.Sprintf(`with rule (%s) test faild. rule runtime options[%#v].test data:(%s). error:(%s)`,
				tester.Rule,
				runtimeOptions,
				tester.Data,
				_err,
			)
			res.Reply = api.ErrCommandReply
			return res, nil
		}

		res.Item = resp.Data
		res.Reply = api.StringCommandReply

	case *command.DropReply:
		if s.scheduler.Exists(&Worker{name: t.Message}) {
			res.Reply = api.ErrCommandReply
			res.Item = fmt.Sprintf("worker (%s) is running.", t.Message)
			break
		}

		_s, err := s.stge.Get(t.Message)
		if err != nil {
			res.Reply = api.ErrCommandReply
			res.Item = fmt.Sprintf("get rule error:(%s) on cache.", t.Message)
			break
		}

		if err := s.stge.Del(t.Message); err != nil {
			log.Debug("drop rule (%s) on cache.\n", t.Message)
			break
		}
		var _err error
		log.Debug("remove all on cache (%s.%s).\n", ll.LIBDB, t.Message)
		dbapi.AccessDatabase(s.persists, ll.LIBDB, t.Message, nil, nil, dbapi.REMOVEALL, s.persistType, &_err)
		if _err != nil {
			log.Error("clean (%s) ns error:%s.\n", t.Message, _err)
			res.Reply = api.ErrCommandReply
			res.Item = _err.Error()
			if err := s.stge.Set(t.Message, _s); err != nil {
				log.Error("rollback on drop rule ns.")
			}
			break
		}
		// clean persists rule
		var _err2 error
		dbapi.AccessDatabase(
			s.persists,
			AUDIT_LOG_DATABASE,
			AUDIT_LOG_RULE,
			bson.M{"_id": t.Message},
			nil,
			dbapi.DEL,
			s.persistType,
			&_err2,
		)
		if _err2 != nil {
			res.Reply = api.ErrCommandReply
			res.Item = _err2.Error()
			log.Error("can not rollback drop rule (%s) error (%s).\n", t.Message, _err2)
			break
		}
		res.Reply = api.OkCommandReply
		_ = _s

	case *command.RunnerReply:
		var _err error
		p := &command.Persist{}
		dbapi.AccessDatabase(
			s.persists,
			AUDIT_LOG_DATABASE,
			AUDIT_LOG_RULE,
			bson.M{"_id": t.Message.Rule},
			p,
			dbapi.GET,
			s.persistType,
			&_err,
		)
		if _err != nil {
			log.Warn("query persists not exists:(%s) error:(%s).\n", t.Message.Rule, _err)
			res.Reply = api.ErrCommandReply
			res.Item = "not persists the rule."
			break
		}
		w := &Worker{
			name:     t.Message.Rule,
			stge:     s.stge,
			persists: s.persists,
		}

		switch t.Message.State {
		case command.START:
			if !s.scheduler.Exists(w) {
				ok := s.scheduler.Add(w)
				if !ok {
					res.Item = fmt.Sprintf("start worker process apply rule (%s) not success.", t.Message.Rule)
					res.Reply = api.ErrCommandReply
					break
				}
				p.Isopen = true
				dbapi.AccessDatabase(
					s.persists,
					AUDIT_LOG_DATABASE,
					AUDIT_LOG_RULE,
					bson.M{"_id": t.Message.Rule},
					p,
					dbapi.SET,
					s.persistType,
					&_err,
				)
				if _err != nil {
					if ok := s.scheduler.Del(w); !ok {
						res.Item = fmt.Sprintf("del (%s) not start success worker error,status brokens.", w.name)
						res.Reply = api.ErrCommandReply
						break
					}
					res.Item = fmt.Sprintf("start worker persists status error(%s).", t.Message.Rule)
					res.Reply = api.ErrCommandReply
					break
				}
				res.Item = fmt.Sprintf("start worker process apply rule (%s).", t.Message.Rule)
				res.Reply = api.StringCommandReply

			} else {
				res.Item = fmt.Sprintf("worker process (%s) already running.", t.Message.Rule)
				res.Reply = api.StringCommandReply
			}

		case command.STOP:
			if s.scheduler.Exists(w) {
				ok := s.scheduler.Del(w)
				if !ok {
					res.Item = fmt.Sprintf("stop worker process rule (%s) not success.", t.Message.Rule)
					res.Reply = api.ErrCommandReply
					break
				}
				p.Isopen = false
				dbapi.AccessDatabase(
					s.persists,
					AUDIT_LOG_DATABASE,
					AUDIT_LOG_RULE,
					bson.M{"_id": t.Message.Rule},
					p,
					dbapi.SET,
					s.persistType,
					&_err,
				)
				if _err != nil {
					res.Item = fmt.Sprintf("stop worker persists status error(%s).", t.Message.Rule)
					res.Reply = api.ErrCommandReply
					break
				}
				res.Item = fmt.Sprintf("stop worker process rule (%s).", t.Message.Rule)
				res.Reply = api.StringCommandReply
				break
			}
			res.Item = fmt.Sprintf("worker process (%s) rule already stop.", t.Message.Rule)
			res.Reply = api.StringCommandReply
		}

	case *command.TopReply:
		ss := s.scheduler.List()
		res.Reply = api.SliceCommandReply
		if len(ss) == 0 {
			res.Items = []string{"not worker running."}
			break
		}
		res.Items = ss

	case *command.ListRuleReply:
		if !s.scheduler.Exists(&Worker{name: t.Message}) {
			res.Reply = api.ErrCommandReply
			res.Item = fmt.Sprintf("not start work %v", t.Message)
			break
		}
		w := s.scheduler.workers[t.Message]
		rs := w.list()
		if len(rs) == 0 {
			res.Reply = api.StringCommandReply
			res.Item = "not monitor file."
			break
		}
		res.Reply = api.SliceCommandReply
		res.Items = rs

	case *command.ErrReply:
		res.Reply = api.ErrCommandReply
		res.Item = fmt.Sprintf("%v", t.Message)

	default:
		return nil, fmt.Errorf("unsupported type %T", reply)
	}

	return res, nil
}

func (s *Server) Run(grpcAddr string) error {
	l, err := net.Listen("tcp", grpcAddr)

	if err != nil {
		return fmt.Errorf("could not listen on %s: %v", grpcAddr, err)
	}

	srv := grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&insecure.Cert)),
		grpc.UnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
	)
	//registry current server
	api.RegisterLogAuditerServer(srv, s)

	return srv.Serve(l)
}
