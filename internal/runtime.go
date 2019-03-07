package internal

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"unsafe"
)

type Marshaler func(interface{}) ([]byte, error)

type Unmarshaler func([]byte, interface{}) error

type ColumnPattern struct {
	DateTime interface{} `bson:"DateTime,omitempty" json:"DateTime,omitempty"`

	IpAddr interface{} `bson:"IpAddr,omitempty" json:"IpAddr,omitempty"`

	State interface{} `bson:"State,omitempty" json:"State,omitempty"`

	UserName interface{} `bson:"UserName,omitempty" json:"UserName,omitempty"`
}

func (c *ColumnPattern) unMarshal(b []byte, aud *AuditLog) {
	rt0 := reflect.ValueOf(aud)
	if rt0.Kind() == reflect.Ptr && !rt0.IsNil() {
		rt0 = rt0.Elem()
	}

	rt := reflect.TypeOf(c).Elem()

	for i := 0; i < rt.NumField(); i++ {
		rv := reflect.Indirect(reflect.ValueOf(c))
		_f := rv.Field(i)

		if !_f.CanInterface() {
			return
		}

		//
		ss := ""
		_interface := _f.Interface()
		switch _interface.(type) {
		case string:
			v, ok := _interface.(string)
			if !ok {
				//
			}
			re, err := regexp.Compile(v)
			if err != nil {
				fmt.Fprintf(os.Stderr, "compile pattern error,colunm=%s,pattern=%s", rt.Field(i).Name, v)
				return
			}
			index := -1
			newB := *(*string)(unsafe.Pointer(&b))
			if newSS := re.FindAllString(newB, index); len(newSS) > 0 {
				ss = newSS[0]
			} else {
				return
			}
		case float64:
			v, ok := _interface.(float64)
			if !ok {
				//
			}

			replaceSpacePattern, err := regexp.Compile(`\ +`)
			if err != nil {
				continue
			}
			ds := *(*string)(unsafe.Pointer(&b))
			newS := replaceSpacePattern.ReplaceAllString(ds, ` `)
			newSS := strings.Split(newS, ` `)
			if len(newSS)+1 < int(v) {
				continue
			}
			ss = newSS[int(v)-1]
		default:
			// 测试
			// fmt.Fprintf(os.Stdout, "[DEBUG] not match rule the interface %v %t.\n", _interface, _interface)
		}

		f := rt0.FieldByName(rt.Field(i).Name)
		if !f.CanSet() {
			//DEBUG
			return
		}
		f.Set(reflect.ValueOf(ss))
	}
}

// 运行时配置文件; ?持久化
type RuntimeOptions struct {
	Host string `bson:"host,omitempty" json:"host,omitempty"`

	LogDate string `bson:"logDate,omitempty" json:"logDate,omitempty"`

	Device string `bson:"device,omitempty" json:"device,omitempty"`
	// system
	SystemType string `bson:"systemType,omitempty" json:"systemType,omitempty"`
	// 日志目录
	Dir string `bson:"dir,omitempty" json:"dir,omitempty"`

	FilePattern string `bson:"filePattern,omitempty" json:"filePattern,omitempty"`

	LinePattern string `bson:"linePattern,omitempty" json:"linePattern,omitempty"`

	LinePrefixPattern string `bson:"linePrefixPattern,omitempty" json:"linePrefixPattern,omitempty"`

	ColumnPattern *ColumnPattern `bson:"columnPattern,omitempty" json:"columnPattern,omitempty"`

	// 日志头格式
	p []byte

	// 日志信息
	o string
}

func (ro *RuntimeOptions) Clone() *RuntimeOptions {
	r := *ro
	return &r
}

func (ro *RuntimeOptions) Marshal(m Marshaler) ([]byte, error) {
	return m(ro)
}

func (ro *RuntimeOptions) Unmarshal(b []byte, m Unmarshaler) error {
	return m(b, ro)
}

func (ro *RuntimeOptions) Handle(data []byte) (res *AuditLog, err error) {
	if len(data) < 1 {
		return nil, errors.New("data is empty.")
	}

	// 匹配行规则
	if linePatternRe, err := regexp.Compile(ro.LinePattern); err != nil {
		return nil, err
	} else if !linePatternRe.Match(data) {
		return nil, errors.New("data not match line pattern.")
	}

	// 处理行头
	var linePrefixError = fmt.Errorf("%s", "data not match line prefix pattern.")

	if linePrefixPatter, err := regexp.Compile(ro.LinePrefixPattern); err != nil {
		return nil, err
	} else if !linePrefixPatter.Match(data) {
		return nil, linePrefixError
	} else {
		d := *(*string)(unsafe.Pointer(&data))
		ss := linePrefixPatter.FindAllString(d, -1)
		if len(ss) > 0 {
			ro.p = []byte(ss[0])
			ro.o = strings.Trim(strings.Replace(d, ss[0], "", -1), " ")
		} else {
			return nil, linePrefixError
		}
	}
	// 处理审计数据表达式
	res = &AuditLog{}
	ro.ColumnPattern.unMarshal(data, res)
	res.Operation, res.SystemType, res.Device = ro.o, ro.SystemType, ro.Device
	return
}

type visitLogAuditParts struct {
	data           []byte
	resp           *Response
	runtimeOptions *RuntimeOptions
	err            *error
}

func newVisitlogAuditParts(data []byte, resp *Response, runtimeOptions *RuntimeOptions, err *error) *visitLogAuditParts {
	vlad := &visitLogAuditParts{
		data:           data,
		resp:           resp,
		runtimeOptions: runtimeOptions,
		err:            err,
	}
	return vlad
}

func (m *visitLogAuditParts) Types() SystemType {
	t := SystemType(strings.ToUpper(m.runtimeOptions.SystemType))
	if "" == t {
		return SystemType(systemName2Type["SERVER"])
	}
	return t
}

func (m *visitLogAuditParts) VisitServerLogs(s *ServerLogs) {
	auditLog, err := m.runtimeOptions.Handle(m.data)
	if err != nil {
		*m.err = err
		return
	}
	s.Content = auditLog
	m.resp.Send(s)
}

func (m *visitLogAuditParts) VisitSwitchLogs(s *SwitchLogs) {
	auditLog, err := m.runtimeOptions.Handle(m.data)
	if err != nil {
		*m.err = err
		return
	}
	s.Content = auditLog
	m.resp.Send(s)
}

func (m *visitLogAuditParts) VisitAppLogs(s *AppLogs) {
	auditLog, err := m.runtimeOptions.Handle(m.data)
	if err != nil {
		*m.err = err
		return
	}
	s.Content = auditLog
	m.resp.Send(s)
}

func VisitLogsAudit2(
	logParts *LogParts,
	data []byte,
	resp *Response,
	runtimeOptions *RuntimeOptions,
	err *error,
) {
	logParts.Accept(newVisitlogAuditParts(data, resp, runtimeOptions, err))
}
