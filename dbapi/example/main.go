package main

import (
	"fmt"
	"logauditer/dbapi"
	"os"

	"gopkg.in/mgo.v2/bson"
)

type x struct {
	Id    string `bson:"_id"`
	Value string `bson:"value"`
}

func main() {
	var err error

	sp := dbapi.NewStorageParts()
	m, err := dbapi.NewKVMongo("localhost:27017")
	if err != nil {
		panic(err)
	}
	sp.AddStoragePart(m)

	var ires1 = &x{"1", "wocao1"}
	var ires2 = &x{"2", "wocao2"}
	var ires3 = &x{"3", "wocao3"}
	var ss []x
	var _err error

	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", bson.M{"_id": ires1.Id}, ires1, dbapi.SET, dbapi.KV, &_err)
	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", bson.M{"_id": ires2.Id}, ires2, dbapi.SET, dbapi.KV, &_err)
	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", bson.M{"_id": ires3.Id}, ires3, dbapi.SET, dbapi.KV, &_err)

	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", "3", nil, dbapi.GET, dbapi.KV, &_err)
	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", "2", nil, dbapi.GET, dbapi.KV, &_err)
	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", "1", nil, dbapi.GET, dbapi.KV, &_err)

	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", nil, &ss, dbapi.KEYS, dbapi.KV, &_err)

	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", "3", nil, dbapi.DEL, dbapi.KV, &_err)
	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", "2", nil, dbapi.DEL, dbapi.KV, &_err)
	dbapi.AccessDatabase(sp, "audit_logs_database", "rule_data", "1", nil, dbapi.DEL, dbapi.KV, &_err)

	dbapi.AccessDatabase(sp, "audit_lib", "yy", nil, nil, dbapi.REMOVEALL, dbapi.KV, &_err)

	dbapi.AccessDatabase(sp, "audit_lib", "yy", nil, nil, dbapi.DROP, dbapi.KV, &_err)
	if _err != nil {
		fmt.Fprintf(os.Stdout, "drop ns error:%s\n", _err)
	}

	var _list = make([]string, 0)
	var _err1 error
	dbapi.AccessDatabase(sp, "logaudit_libdb", "", nil, &_list, dbapi.LIST, dbapi.KV, &_err1)

	if _err1 != nil {
		fmt.Fprintf(os.Stderr, "err:(%s)\n", _err1)
	}

	for _, _x := range _list {
		fmt.Fprintf(os.Stdout, "collection = %#v\n", _x)
	}
	var test = struct {
		Id   string
		Name string
	}{
		"123",
		"wocao",
	}
	dbapi.AccessDatabase(sp, "test", "test1", bson.M{"_id": "123"}, &test, dbapi.SET, dbapi.KV, &_err)

	var ret error
	dbapi.AccessDatabase(sp, "test", "test1", nil, ret, dbapi.MV, dbapi.KV, &_err)
	if _err != nil {
		fmt.Fprintf(os.Stderr, "mv error %s\n", _err)
	}

	fmt.Fprintf(os.Stdout, "ss = %#v\n", ss)
}
