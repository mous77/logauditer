package internal

import "fmt"

// client request channel response value.

type Response struct {
	Data string
	Err  error
}

func NewResponse() *Response {
	return &Response{}
}

func (r *Response) Send(v VisitorLogAudit) {
	vstr := v.String()
	if len(vstr) == 0 {
		r.Err = fmt.Errorf("empty data.")
	}
	r.Data = vstr
}
