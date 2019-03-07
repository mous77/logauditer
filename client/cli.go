package client

import (
	"context"
	"fmt"
	"logauditer/api"
	"logauditer/insecure"
	"logauditer/raw"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Bowery/prompt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const connectTimeout = 200 * time.Millisecond

const prefix = ">"

var prmpt = ""

//CLI allows users to interact with a server.
type CLI struct {
	printer *printer
	term    *prompt.Terminal
	conn    *grpc.ClientConn
	client  api.LogAuditerClient
}

//Run runs a new CLI.
func Run(hostPorts string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		hostPorts,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
	)
	if err != nil {
		return fmt.Errorf("could not dial %s: %v", hostPorts, err)
	}

	term, err := prompt.NewTerminal()
	if err != nil {
		return fmt.Errorf("could not create a terminal: %v", err)
	}

	prmpt = fmt.Sprintf("%s%s", "", prefix)

	c := &CLI{
		printer: newPrinter(os.Stdout),
		term:    term,
		client:  api.NewLogAuditerClient(conn),
		conn:    conn,
	}

	defer func() {
		err = c.Close()
	}()

	c.run()

	return nil
}

//Close closes the CLI.
func (c *CLI) Close() error {
	if err := c.printer.Close(); err != nil {
		return err
	}
	if err := c.conn.Close(); err != nil {
		return err
	}
	return c.term.Close()
}

func (c *CLI) run() {
	c.printer.printLogo()
	cb := newCommandBuffer()
	for {
		input, err := c.term.GetPrompt(prmpt)

		if err != nil {
			if err == prompt.ErrCTRLC || err == prompt.ErrEOF {
				break
			}
			c.printer.printError(err)
			continue
		}
		if input == "" {
			continue
		}

		if input[len(input)-1] != ';' {
			cb.add(input)
			prmpt = fmt.Sprintf("%s%s", " ", "...")
			continue
		}

		cb.add(input)
		command := cb.clean()
		req := &api.ExecuteRequest{Command: raw.Raw(command)}
		prmpt = fmt.Sprintf("%s%s", "", prefix)
		if resp, err := c.client.Execute(context.Background(), req); err != nil {
			c.printer.printError(err)
		} else {
			c.printer.printResponse(resp)
		}

	}
	c.printer.println("Bye!")
}

type commandBuffer struct {
	mu sync.Mutex
	bb []string
}

func newCommandBuffer() *commandBuffer {
	return &commandBuffer{
		mu: sync.Mutex{},
		bb: make([]string, 0),
	}
}

func (c *commandBuffer) add(x string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bb = append(c.bb, x)
}

func (c *commandBuffer) clean() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	ss := strings.Join(c.bb, "")
	c.bb = c.bb[:0]
	ss = strings.TrimSuffix(ss, ";")
	// fmt.Fprintf(os.Stdout, "command=%s ", ss)
	return ss
}
