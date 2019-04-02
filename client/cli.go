package client

import (
	"context"
	"fmt"
	"logauditer/api"
	"logauditer/insecure"
	"logauditer/raw"
	"os"
	"time"

	"github.com/laik/prompt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const connectTimeout = 200 * time.Millisecond

const prefix = "> "

var prmpt = ""

//CLI allows users to interact with a server.
type CLI struct {
	printer *printer
	term    *prompt.Prompt
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

	term := prompt.NewPrompt()
	if err != nil {
		return fmt.Errorf("could not create a terminal: %v", err)
	}

	prmpt = fmt.Sprintf("%s%s", "", prefix)

	term.SetPrefix(prmpt)

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
	return nil
}

func (c *CLI) run() {
	c.printer.printLogo()

	h := func(command string) {
		req := &api.ExecuteRequest{Command: raw.Raw(command)}
		prmpt = fmt.Sprintf("%s%s", "", prefix)
		if resp, err := c.client.Execute(context.Background(), req); err != nil {
			c.printer.printError(err)
		} else {
			c.printer.printResponse(resp)
		}
	}

	c.term.Handler(h)

	c.printer.println("Bye!")
}
