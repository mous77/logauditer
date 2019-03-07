package dbapi

import (
	"time"

	"github.com/globalsign/mgo"
)

type KeyValueApi struct {
	t  DBType
	DB *mgo.Session
}

func NewKVMongo(url string) (*KeyValueApi, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	// maximum pooled connections. the overall established sockets
	// should be lower than this value(will block otherwise)
	session.SetPoolLimit(256)
	session.SetSocketTimeout(10 * time.Minute)

	if err := session.Ping(); err != nil {
		return nil, err
	}

	session.SetMode(mgo.Primary, true)

	return &KeyValueApi{
		t:  KV,
		DB: session,
	}, nil
}

func (a *KeyValueApi) Accept(h DBApiHandler) {
	defaultH(h, a)
}

func (a *KeyValueApi) Get() interface{} {
	return a.DB
}

func (a *KeyValueApi) T() DBType {
	return a.t
}

func (a *KeyValueApi) C() {
	a.DB.Close()
}
