package internal

import (
	"errors"
)

var NOT_MATCH_PREFIX = errors.New("can't match prefix msg format.")

type Rule interface{}

// Interface of the visitor
type LogVisitor interface {
	Types() SystemType
	VisitAppLogs(*AppLogs)
	VisitServerLogs(*ServerLogs)
	VisitSwitchLogs(*SwitchLogs)
}

// multiple log parts
type VisitorLogAudit interface {
	Accept(LogVisitor)
	String() string
}

// Part
// 新增加审计日志格式需要添加对应的struct
// 日志转换
type LogParts struct {
	parts map[SystemType]VisitorLogAudit
}

func NewLogParts() *LogParts {
	lp := &LogParts{
		parts: map[SystemType]VisitorLogAudit{
			SERVER: &ServerLogs{},
			SWITCH: &SwitchLogs{},
			APP:    &AppLogs{},
		},
	}
	return lp
}

func (this *LogParts) Accept(visitor LogVisitor) {
	stp := visitor.Types()
	if part, ok := this.parts[stp]; ok {
		part.Accept(visitor)
	}
}
