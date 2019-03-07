package dbapi

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type RDBMSApi struct {
	t  DBType
	DB *sql.DB
}

func NewRDBMSMysql(url string) (*RDBMSApi, error) {
	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, err
	}
	return &RDBMSApi{
		t:  RDBMS,
		DB: db,
	}, nil
}

func (a *RDBMSApi) Accept(h DBApiHandler) {
	defaultH(h, a)
}

func (a *RDBMSApi) Get() interface{} {
	return a.DB
}

func (a *RDBMSApi) T() DBType {
	return a.t
}

func (a *RDBMSApi) C() {
	a.DB.Close()
}
