package command

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"
)

//ErrCommandNotFound means that command could not be parsed.
var ErrCommandNotFound = errors.New("command: not found")

//Parser is a parser that parses user input and creates the appropriate command.
type Parser struct {
	stge DataStore
}

//NewParser creates a new parser
func NewParser(stge DataStore) *Parser {
	return &Parser{stge: stge}
}

//Parse parses string to Command with args
func (p *Parser) Parse(str string) (Command, []string, error) {
	var cmd Command
	args := p.extractArgs(str)

	if len(args) == 0 {
		fmt.Fprintf(os.Stdout, "[DEBUG] str (%v) args length 0\n", str)
		return nil, nil, ErrCommandNotFound
	}

	switch strings.ToUpper(args[0]) {
	case "HELP":
		cmd = &Help{parser: p}
	case "DESC":
		cmd = &Get{stge: p.stge}
	case "SET":
		cmd = &Set{stge: p.stge}
	case "RULES":
		cmd = &Keys{stge: p.stge}
	case "COMMIT":
		cmd = &Commit{stge: p.stge}
	case "TEST":
		cmd = &Test{stge: p.stge}
	case "START":
		cmd = &Start{stge: p.stge}
	case "STOP":
		cmd = &Stop{stge: p.stge}
	case "DROP":
		cmd = &Drop{stge: p.stge}
	case "TOP":
		cmd = &Top{}
	case "LIST":
		cmd = &List{stge: p.stge}
	default:
		return nil, nil, ErrCommandNotFound
	}

	return cmd, args[1:], nil
}

func (p *Parser) extractArgs(val string) []string {
	args := make([]string, 0)
	var inQuote bool
	var buf bytes.Buffer
	for _, r := range val {
		switch {
		case r == '`':
			inQuote = !inQuote
		case unicode.IsSpace(r):
			if !inQuote && buf.Len() > 0 {
				args = append(args, buf.String())
				buf.Reset()
			} else {
				buf.WriteRune(r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		args = append(args, buf.String())
	}
	return args
}
