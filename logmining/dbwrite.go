package logmining

import (
	"encoding/json"
	"logauditer/dbapi"
	in "logauditer/internal"
	"strings"
	"time"
)

const LOG_RECORD = "logrecord"

type DBWrite struct {
	sp          *dbapi.StorageParts
	Database    string
	Collections string
}

func NewDBWrite(sp *dbapi.StorageParts) *DBWrite {
	dbw := &DBWrite{
		sp: sp,
	}
	dbw.updateCollection()
	return dbw
}

func (d *DBWrite) Write(host string, date string, data []byte) error {
	res := &in.AuditLog{}
	if err := json.Unmarshal(data, res); err != nil {
		return err
	}
	res.Host = host
	res.Date = date
	var _err error
	dbapi.AccessDatabase(
		d.sp,
		d.Database,
		d.Collections,
		nil,
		res,
		dbapi.INSERT,
		dbapi.KV,
		&_err,
	)
	return _err
}

func (d *DBWrite) updateCollection() {
	d.Database = LOG_RECORD
	gencoll := func() {
		timeLayout := "2006 01 02" //
		source := time.Unix(time.Now().Unix(), 0).Format(timeLayout)
		d.Collections = "log_" + strings.Replace(source, " ", "_", 2)
	}
	gencoll()
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				gencoll()
			}
		}
	}()
}
