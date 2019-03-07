package dbapi

import (
	"database/sql"
	"errors"
	"strings"
	"sync"

	"github.com/globalsign/mgo/bson"

	"github.com/globalsign/mgo"
)

type OPType uint

const (
	GET OPType = iota
	SET
	DEL
	KEYS
	INSERT
	DROP
	LIST
	MV
	REMOVEALL
)

type DBType interface{}

var (
	// db
	KV    = &mgo.Session{}
	RDBMS = &sql.DB{}
)

type DBApiHandler interface {
	Get(Storager)
	Set(Storager)
	Del(Storager)
	Keys(Storager)
	Insert(Storager)
	List(Storager)
	Drop(Storager)
	Mv(Storager)
	RemoveAll(Storager)
	Op() OPType
	Types() DBType
}

type Storager interface {
	Accept(DBApiHandler)
	Get() interface{}
	T() DBType
	C()
}

type StorageParts struct {
	//
	mu sync.Mutex

	parts map[DBType]Storager
}

func (s *StorageParts) AddStoragePart(part Storager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parts[part.T()] = part
}

func (s *StorageParts) DropStoragePart(part Storager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := part.T()
	s.parts[t].C()
	delete(s.parts, t)
}

func NewStorageParts() *StorageParts {
	parts := make(map[DBType]Storager)
	return &StorageParts{
		mu:    sync.Mutex{},
		parts: parts,
	}
}

func (s *StorageParts) Accept(h DBApiHandler) {
	t := h.Types()
	_, ok := s.parts[t]
	if !ok {
		return
	}

	part := s.parts[t]
	var x = func(t OPType) func(Storager) {
		switch t {
		case GET:
			return h.Get
		case SET:
			return h.Set
		case DEL:
			return h.Del
		case KEYS:
			return h.Keys
		case INSERT:
			return h.Insert
		case DROP:
			return h.Drop
		case LIST:
			return h.List
		case REMOVEALL:
			return h.RemoveAll
		default:
			return nil
		}
	}

	if f := x(h.Op()); f != nil {
		f(part)
	}
}

func defaultH(h DBApiHandler, s Storager) {
	switch h.Op() {
	case GET:
		h.Get(s)
	case SET:
		h.Set(s)
	case DEL:
		h.Del(s)
	case KEYS:
		h.Keys(s)
	case INSERT:
		h.Insert(s)
	case DROP:
		h.Drop(s)
	case LIST:
		h.List(s)
	case MV:
		h.Mv(s)
	case REMOVEALL:
		h.RemoveAll(s)
	}
}

type DBApiRequestMessage struct {
	DB    string
	Table string
	Query interface{}
	Res   interface{}
	Err   *error
	Otyp  OPType
	Dtyp  DBType
}

func (a *DBApiRequestMessage) cursor(s Storager) *mgo.Collection {
	session := s.Get()
	if _, ok := session.(*mgo.Session); !ok {
		return nil
	}
	ss := session.(*mgo.Session)
	return ss.DB(a.DB).C(a.Table)
}

func (a *DBApiRequestMessage) Get(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		*a.Err = coll.Find(a.Query).One(a.Res)
	default:
	}
}

func (a *DBApiRequestMessage) Set(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		_, *a.Err = coll.Upsert(a.Query, a.Res)

	default:
	}
}

func (a *DBApiRequestMessage) Del(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		*a.Err = coll.Remove(a.Query)
	default:
	}
}

func (a *DBApiRequestMessage) Keys(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		*a.Err = coll.Find(a.Query).All(a.Res)
	default:
	}
}

func (a *DBApiRequestMessage) Insert(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		*a.Err = coll.Insert(a.Res)
	default:
	}
}

func (a *DBApiRequestMessage) RemoveAll(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		_, *a.Err = coll.RemoveAll(bson.M{})
	default:
	}
}

func (a *DBApiRequestMessage) Drop(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		coll := a.cursor(s)
		*a.Err = coll.DropCollection()
	default:
	}
}

func (a *DBApiRequestMessage) Mv(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		session := s.Get()
		ss, ok := session.(*mgo.Session)
		if !ok {
			return
		}
		source := strings.Join([]string{a.DB, a.Table}, ".")
		target := strings.Join([]string{a.DB, a.Table + "_bak"}, ".")
		*a.Err = ss.Run(
			bson.D{
				bson.DocElem{Name: "renameCollection", Value: source},
				bson.DocElem{Name: "to", Value: target},
			},
			a.Res,
		)
	default:
	}
}
func (a *DBApiRequestMessage) List(s Storager) {
	switch a.Dtyp {
	case RDBMS:
		//
	case KV:
		session := s.Get()
		ss, ok := session.(*mgo.Session)
		if !ok {
			return
		}
		names, err := ss.DB(a.DB).CollectionNames()
		if err != nil {
			*a.Err = err
		}
		r, ok := a.Res.(*[]string)
		if !ok {
			*a.Err = errors.New("result is not *[]string type.")
			return
		}
		for _, n := range names {
			*r = append(*r, n)
		}
	default:
	}
}
func (a *DBApiRequestMessage) Op() OPType { return a.Otyp }

func (a *DBApiRequestMessage) Types() DBType { return a.Dtyp }

func AccessDatabase(
	sp *StorageParts,
	db, table string,
	query interface{},
	res interface{},
	o OPType,
	t DBType,
	err *error,
) {
	msg := &DBApiRequestMessage{
		DB:    db,
		Table: table,
		Query: query,
		Err:   err,
		Res:   res,
		Otyp:  o,
		Dtyp:  t,
	}
	sp.Accept(msg)
}
