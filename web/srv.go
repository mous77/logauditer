package web

import (
	"encoding/json"
	"fmt"
	"logauditer/dbapi"
	"logauditer/internal"
	"net/http"
	"net/url"
	"strings"

	ll "logauditer/logmining"

	"github.com/globalsign/mgo/bson"
	log "github.com/laik/logger"
)

func NewHttpServer(addr string, httpSrv *HttpService) {

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", srvHome)

	// http://127.0.0.1/getInfo?ipaddr=10.10.2.104&date=2019-02-26
	http.HandleFunc("/getInfo",
		func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			if res, err := httpSrv.Query(r.Form); err != nil {
				fmt.Fprintf(w, "%s", err)
			} else {
				bytes, err := json.Marshal(res)
				if err != nil {
					log.Error("unmarshal query result error:(%s).\n", err)
					return
				}
				fmt.Fprintf(w, "%s", bytes)
			}
		},
	)

	log.Info("start http server %s.\n", addr)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe address (%s) error(%s): ", addr, err)
	}
}

func srvHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	http.ServeFile(w, r, "home.html")
}

type Bean struct {
	Ipaddr string `json:"ipaddr"`
	Date   string `json:"date"`
	Type   string `json:"type"`
}

func NewBean() *Bean {
	return &Bean{}
}

type Result []internal.AuditLog

type HttpService struct {
	SP *dbapi.StorageParts
}

func (h *HttpService) Query(form url.Values) (*Result, error) {
	query := bson.M{}

	var collectionName string

	if ipaddr := form.Get("ipaddr"); ipaddr != "" {
		query["Host"] = ipaddr
		log.Debug("i except get ipaddr (%s).\n", ipaddr)
	}
	if date := form.Get("date"); date != "" {
		query["Date"] = date
		collectionName = "log_" + strings.Replace(date, "-", "_", 2)
	}
	if _type := form.Get("type"); _type != "" {
		query["SystemType"] = _type
	}
	res := new(Result)
	var _err error
	dbapi.AccessDatabase(
		h.SP,
		ll.LOG_RECORD,
		collectionName,
		query,
		res,
		dbapi.KEYS,
		dbapi.KV,
		&_err,
	)
	if _err != nil {
		return nil, _err
	}
	return res, nil
}
