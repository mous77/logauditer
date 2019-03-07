package internal

import "unsafe"

type ServerLogs struct {
	Content *AuditLog
	// Extend info
}

func (this *ServerLogs) Accept(visitor LogVisitor) {
	if this.Content == nil {
		this.Content = &AuditLog{}
	}
	visitor.VisitServerLogs(this)
}

func (this *ServerLogs) String() string {
	b, err := Marshal(this.Content)
	if err != nil {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}

type SwitchLogs struct {
	Content *AuditLog
}

func (this *SwitchLogs) Accept(visitor LogVisitor) {
	if this.Content == nil {
		this.Content = &AuditLog{}
	}
	visitor.VisitSwitchLogs(this)
}

func (this *SwitchLogs) String() string {
	b, err := Marshal(this.Content)
	if err != nil {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}

type AppLogs struct {
	Content *AuditLog
}

func (this *AppLogs) Accept(visitor LogVisitor) {
	if this.Content == nil {
		this.Content = &AuditLog{}
	}
	visitor.VisitAppLogs(this)
}

func (this *AppLogs) String() string {
	b, err := Marshal(this.Content)
	if err != nil {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}
