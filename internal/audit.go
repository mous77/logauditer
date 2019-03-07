package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var (
	INVALID_ERR   = errors.New("new log with invalid data stream.")
	UNMARSHAL_ERR = errors.New("unmarshal data error.")
)

type ColumnTypeName string

type SystemType string

// 分割字段名
const (
	UserName  = `UserName`
	IpAddr    = `IpAddr`
	State     = `State`
	DateTime  = `DateTime`
	Operation = `Operation`
)

var colTyp2NameMap = map[string]string{
	UserName:  "UserName",
	IpAddr:    "Ipaddr",
	State:     "State",
	DateTime:  "DateTime",
	Operation: "Operation",
}

var colName2TypMap = map[string]string{
	"UserName":  UserName,
	"IpAddr":    IpAddr,
	"State":     State,
	"DateTime":  DateTime,
	"Operation": Operation,
}

// 系统类型
const (
	SERVER = "SERVER"
	SWITCH = "SWITCH"
	APP    = "APP"
)

var systemType2Name = map[string]string{
	"SERVER": SERVER,
	"SWITCH": SWITCH,
	"APP":    APP,
}

var systemName2Type = map[string]string{
	SERVER: "SERVER",
	SWITCH: "SWITCH",
	APP:    "APP",
}

// 命令相关操作信息
type AuditLog struct {
	Host string `bson:"Host,omitempty" json:"Host,omitempty"`
	// 日志产生日期
	Date string `bson:"Date,omitempty" json:"Date,omitempty"`

	// device 设备类型【服务器，交换机，服务器x86-64，powerPC,Sun....】
	Device string `bson:"Device,omitempty" json:"Device,omitempty"`
	// 系统类型
	SystemType string `bson:"SystemType,omitempty" json:"SystemType,omitempty"`

	DateTime  string `bson:"DateTime" json:"DateTime"`
	IpAddr    string `bson:"IpAddr,omitempty" json:"IpAddr,omitempty"`
	Operation string `bson:"Operation,omitempty" json:"Operation,omitempty"`
	State     string `bson:"State,omitempty" json:"State,omitempty"`
	UserName  string `bson:"UserName,omitempty" json:"UserName,omitempty"`
}

func Marshal(auditlog *AuditLog) (b []byte, err error) {
	b, err = json.Marshal(auditlog)
	if err != nil {
		return
	}
	return
}

func UnMarshal(data []byte, auditLog *AuditLog) error {
	if data == nil || len(data) < 1 {
		return INVALID_ERR
	}
	if err := json.Unmarshal(data, auditLog); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] unmarshal error: %s\n", err)
		return UNMARSHAL_ERR
	}
	return nil
}
