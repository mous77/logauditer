package client

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"logauditer/api"

	"github.com/fatih/color"
)

const (
	okString  = "OK"
	nilString = "(nil)"
)

const logo = `
logAuditer Terminal
`

type printer struct {
	okColor  *color.Color
	errColor *color.Color
	nilColor *color.Color
	out      io.Writer
}

func newPrinter(out io.Writer) *printer {
	return &printer{
		okColor:  color.New(color.FgHiGreen),
		errColor: color.New(color.FgHiRed),
		nilColor: color.New(color.FgHiCyan),
		out:      out,
	}
}

//Close closes the printer
func (p *printer) Close() error {
	if cl, ok := p.out.(io.Closer); ok {
		return cl.Close()
	}
	return nil
}

func (p *printer) printLogo() {
	color.Set(color.FgMagenta)
	p.println(strings.Replace(logo, "\n", "\r\n", -1))
	color.Unset()
}

func (p *printer) println(str string) {
	fmt.Fprintf(p.out, "%s\r\n", str)
}

func (p *printer) printError(err error) {
	p.errColor.Fprintf(p.out, "Error: %s\n", err.Error())
}

func (p *printer) printResponse(resp *api.ExecuteCommandResponse) {
	switch resp.Reply {
	case api.OkCommandReply:
		p.println(p.okColor.Sprint(okString))
	case api.NilCommandReply:
		p.println(p.nilColor.Sprint(nilString))
	case api.StringCommandReply:
		p.println(fmt.Sprintf("S| %s", resp.Item))
	case api.ErrCommandReply:
		p.println(p.errColor.Sprintf("E| %s", resp.Item))
	case api.SliceCommandReply:
		items := resp.Items
		buf := new(bytes.Buffer)
		for i, v := range resp.Items {
			buf.WriteString(fmt.Sprintf("%d| %q", i+1, v))
			// check whether the current item is not last
			if i != len(items)-1 {
				buf.WriteString("\r\n")
			}
		}
		p.println(buf.String())
	default:
		fmt.Fprintf(p.out, "%v\n", resp)
	}
}
