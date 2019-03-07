package command

type Reply interface {
	Val() interface{}
}

type ErrReply struct {
	Message error
}

func (this *ErrReply) Val() interface{} {
	return this.Message
}

type OkReply struct{}

func (this *OkReply) Val() interface{} {
	return nil
}

type StringReply struct {
	Message string
}

func (this *StringReply) Val() interface{} {
	return this.Message
}

type NilReply struct{}

func (this *NilReply) Val() interface{} {
	return nil
}

type SliceReply struct {
	Message []string
}

func (this *SliceReply) Val() interface{} {
	return this.Message
}

type Persist struct {
	Id     string `bson:"_id,omitempty" json:"_id,omitempty"`
	Value  string `bson:"value,omitempty" json:"value,omitempty"`
	Isopen bool   `bson:"isopen,omitempty" json:"isopen,omitempty" `
}

type PersistReply struct {
	Message Persist
}

func (this *PersistReply) Val() interface{} {
	return this.Message
}

type TestReply struct {
	Message Tester
}

func (this *TestReply) Val() interface{} {
	return this.Message
}

type Tester struct {
	Rule string `bson:"rule,omitempty" json:"rule,omitempty"`
	Data []byte `bson:"data,omitempty" json:"data,omitempty"`
}

type RunnerState struct {
	Rule  string `bson:"rule,omitempty" json:"rule,omitempty"`
	State int    `bson:"state,omitempty" json:"state,omitempty"`
}

type RunnerReply struct {
	Message RunnerState
}

func (this *RunnerReply) Val() interface{} {
	return this.Message
}

type DropReply struct {
	Message string
}

func (this *DropReply) Val() interface{} {
	return this.Message
}

type TopReply struct{}

func (this *TopReply) Val() interface{} { return nil }

type ListRuleReply struct {
	Message string
}

func (this ListRuleReply) Val() interface{} { return this.Message }
