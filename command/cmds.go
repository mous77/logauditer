package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"logauditer/internal"
	"os"
	"regexp"
	"strings"
)

const (
	START = iota
	STOP
)

type Test struct {
	stge DataStore
}

func (this *Test) Name() string {
	return "TEST"
}

func (this *Test) Help() string {
	return `TEST ${DATA} WITH ${RULE_NAME}`
}

func (this *Test) Execute(args ...string) Reply {
	if len(args) != 3 {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	if strings.ToLower(args[1]) != "with" {
		return &ErrReply{Message: errors.New(this.Help())}
	}
	rule, err := this.stge.Get(args[2])
	if err != nil {
		return &ErrReply{Message: err}
	}
	return &TestReply{Message: Tester{Rule: rule, Data: []byte(args[0])}}
}

type Commit struct {
	stge DataStore
}

func (this *Commit) Name() string {
	return "COMMIT"
}

func (this *Commit) Help() string {
	return "COMMIT ${RULE_NAME}"
}

func (this *Commit) Execute(args ...string) Reply {
	if len(args) != 1 {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	value, err := this.stge.Get(args[0])
	if err != nil {
		return &ErrReply{Message: errors.New("not found rule in cache.")}
	}
	p := Persist{Id: args[0], Value: value}

	return &PersistReply{Message: p}
}

type Set struct {
	stge DataStore
}

//Name returns the command name.
func (this *Set) Name() string {
	return "SET ${RULE_NAME} ${DATA}"
}

//Help returns information about the command. Description, usage and etc.
func (this *Set) Help() string {
	return `Usage: SET ${RULE_NAME} "${DATA}"`
}

//Execute executes the command with the given arguments.
func (this *Set) Execute(args ...string) Reply {
	if len(args) < 2 {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	runtimeOps := internal.RuntimeOptions{}
	err := runtimeOps.Unmarshal([]byte(args[1]), json.Unmarshal)
	if err != nil {
		return &ErrReply{Message: fmt.Errorf("rule unmarshal err: %s", err)}
	}
	err = this.stge.Set(args[0], args[1])
	if err != nil {
		return &ErrReply{Message: err}
	}
	return &OkReply{}
}

type Get struct {
	stge DataStore
}

//Name returns the command name.
func (this *Get) Name() string {
	return "GET"
}

//Help returns information about the command. Description, usage and etc.
func (this *Get) Help() string {
	return `Usage: Get ${RULE_NAME}`
}

//Execute executes the command with the given arguments.
func (this *Get) Execute(args ...string) Reply {
	var reply Reply

	reply = checkExpcetArgs(1, args...)
	if _, ok := reply.(*ErrReply); ok {
		return reply
	}

	value, err := this.stge.Get(args[0])

	if err != nil {
		reply = &ErrReply{Message: err}
	}
	reply = &StringReply{
		Message: value,
	}
	return reply
}

type Keys struct {
	stge DataStore
}

//Name returns the command name.
func (this *Keys) Name() string {
	return "KEYS"
}

//Help returns information about the command. Description, usage and etc.
func (this *Keys) Help() string {
	return `Usage: KEYS`
}

//Execute executes the command with the given arguments.
func (this *Keys) Execute(args ...string) Reply {
	var (
		err     error
		pattern *regexp.Regexp
	)
	if len(args) > 1 {
		return &ErrReply{Message: ErrWrongArgsNumber}
	}
	if len(args) == 1 {
		pattern, err = regexp.CompilePOSIX(args[0])
		if err != nil {
			return &ErrReply{Message: err}
		}
	}

	keysIter, err := this.stge.Keys()
	if err != nil {
		return &ErrReply{Message: err}
	}
	keysName := make([]string, 0, len(keysIter))
	for _, k := range keysIter {
		if pattern != nil {
			if !pattern.MatchString(k) {
				fmt.Fprintf(os.Stdout, "[DEBUG] regexp compile posix is not match %s %v\n", args[0], pattern)
				continue
			}
		}
		keysName = append(keysName, k)
	}
	return &SliceReply{Message: keysName}
}

//Help is the Help command
type Help struct {
	parser CommandParser
}

//Name implements Name of Command interface
func (c *Help) Name() string {
	return "HELP"
}

//Help implements Help of Command interface
func (c *Help) Help() string {
	return `Usage: HELP command [Show the usage of the given command]`
}

//Execute implements Execute of Command interface
func (c *Help) Execute(args ...string) Reply {
	var reply Reply
	switch len(args) {
	case 1:
		reply = &StringReply{Message: c.Help()}
	case 2:
		cmdName := args[0]
		cmd, _, err := c.parser.Parse(cmdName)
		if err != nil {
			if err == ErrCommandNotFound {
				return &ErrReply{Message: fmt.Errorf("command %q not found", cmdName)}
			}
			return &ErrReply{Message: err}
		}
		reply = &StringReply{Message: cmd.Help()}
	default:
		helpErr := fmt.Errorf("%s: %s. arguments length %d", ErrWrongArgsNumber, c.Help(), len(args))
		reply = &ErrReply{Message: helpErr}
	}
	return reply
}

type Start struct {
	stge DataStore
}

func (this *Start) Name() string {
	return "START"
}

func (this *Start) Help() string {
	return `Usage: START ${RULE_NAME}`
}

func (this *Start) Execute(args ...string) Reply {
	if reply, ok := checkExpcetArgs(1, args...).(*ErrReply); ok {
		return reply
	}
	_, err := this.stge.Get(args[0])
	if err != nil {
		return &ErrReply{
			Message: fmt.Errorf("not define rule (%s).", args[0]),
		}
	}
	return &RunnerReply{
		RunnerState{
			Rule:  args[0],
			State: START,
		},
	}
}

type Stop struct {
	stge DataStore
}

func (this *Stop) Name() string {
	return "STOP"
}

func (this *Stop) Help() string {
	return `Usage: STOP ${RULE_NAME}`
}

func (this *Stop) Execute(args ...string) Reply {
	if reply, ok := checkExpcetArgs(1, args...).(*ErrReply); ok {
		return reply
	}
	_, err := this.stge.Get(args[0])
	if err != nil {
		return &ErrReply{
			Message: fmt.Errorf("not define rule (%s).", args[0]),
		}
	}
	return &RunnerReply{
		RunnerState{
			Rule:  args[0],
			State: STOP,
		},
	}
}

type Drop struct {
	stge DataStore
}

func (this *Drop) Name() string {
	return "DROP"
}

func (this *Drop) Help() string {
	return `Usage: DROP ${RULE_NAME}`
}

func (this *Drop) Execute(args ...string) Reply {
	if reply, ok := checkExpcetArgs(1, args...).(*ErrReply); ok {
		return reply
	}
	return &DropReply{Message: args[0]}
}

type Top struct{}

func (this *Top) Name() string {
	return "TOP"
}

func (this *Top) Help() string {
	return `Usage: TOP`
}

func (this *Top) Execute(args ...string) Reply {
	if reply, ok := checkExpcetArgs(0, args...).(*ErrReply); ok {
		return reply
	}
	return &TopReply{}
}

type List struct {
	stge DataStore
}

func (this *List) Name() string {
	return "LIST"
}
func (this *List) Help() string {
	return `Usage: List #{RULE_NAME}`
}

func (this *List) Execute(args ...string) Reply {
	if reply, ok := checkExpcetArgs(1, args...).(*ErrReply); ok {
		return reply
	}
	if _, err := this.stge.Get(args[0]); err != nil {
		return &ErrReply{Message: fmt.Errorf("rule is not exists.")}
	}
	return &ListRuleReply{Message: args[0]}
}
