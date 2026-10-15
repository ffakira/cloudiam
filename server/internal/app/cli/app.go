package cliapp

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	authv1 "github.com/ffakira/cloudiam/server/gen/auth/v1"
	cliadapter "github.com/ffakira/cloudiam/server/internal/adapters/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const cloudIAMInternalAddr = "127.0.0.1:50051"

type requestCommand struct {
	Target          string
	DurationSeconds int32
	OutputFormat    string
}

func Run(ctx context.Context, args []string, stdout io.Writer) error {
	command, err := parseCommand(args)
	if err != nil {
		return err
	}

	conn, err := grpc.NewClient(cloudIAMInternalAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create grpc client: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	switch command.Name {
	case "login":
		sessionPath, err := cliadapter.DefaultSessionPath()
		if err != nil {
			return err
		}
		return cliadapter.Login(ctx, authv1.NewAuthServiceClient(conn), cliadapter.NewSessionStore(sessionPath), cliadapter.LoginInput{}, stdout)
	case "access request":
		request := command.Request
		sessionPath, err := cliadapter.DefaultSessionPath()
		if err != nil {
			return err
		}
		session, err := cliadapter.NewSessionStore(sessionPath).Load()
		if err != nil {
			return err
		}
		client := accessv1.NewAccessServiceClient(conn)
		return cliadapter.RequestAccess(ctx, client, cliadapter.RequestAccessInput{
			CloudIAMSessionToken: session.AccessToken,
			Target:               request.Target,
			DurationSeconds:      request.DurationSeconds,
			OutputFormat:         request.OutputFormat,
		}, stdout)
	default:
		return fmt.Errorf("unknown command %q", command.Name)
	}
}

type command struct {
	Name    string
	Request requestCommand
}

func parseCommand(args []string) (command, error) {
	fs := flag.NewFlagSet("access-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return command{}, err
	}
	if fs.NArg() == 0 {
		return command{}, fmt.Errorf("command is required")
	}
	if fs.Arg(0) == "login" {
		if fs.NArg() != 1 {
			return command{}, fmt.Errorf("login does not accept arguments")
		}
		return command{Name: "login"}, nil
	}
	request, err := parseRequestCommand(args)
	if err != nil {
		return command{}, err
	}
	return command{Name: "access request", Request: request}, nil
}

func parseRequestCommand(args []string) (requestCommand, error) {
	fs := flag.NewFlagSet("access-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return requestCommand{}, err
	}

	if fs.NArg() < 2 {
		return requestCommand{}, fmt.Errorf("command is required")
	}
	if fs.Arg(0) != "access" || fs.Arg(1) != "request" {
		return requestCommand{}, fmt.Errorf("unknown command %q", fs.Arg(0))
	}

	cmd := flag.NewFlagSet("access request", flag.ContinueOnError)
	cmd.SetOutput(io.Discard)
	duration := cmd.Int("duration", 0, "duration in seconds")
	format := cmd.String("format", "summary", "output format: summary or env")
	cmdArgs := fs.Args()[2:]
	if len(cmdArgs) == 0 {
		return requestCommand{}, fmt.Errorf("target is required")
	}
	target := cmdArgs[0]
	if target == "" || target[0] == '-' {
		return requestCommand{}, fmt.Errorf("target is required")
	}
	if err := cmd.Parse(cmdArgs[1:]); err != nil {
		return requestCommand{}, err
	}
	if cmd.NArg() != 0 {
		return requestCommand{}, fmt.Errorf("access request accepts exactly one target")
	}
	if *duration < 0 {
		return requestCommand{}, fmt.Errorf("duration must be non-negative")
	}
	if int64(*duration) > maxInt32 {
		return requestCommand{}, fmt.Errorf("duration must fit int32 seconds")
	}
	if *format != "summary" && *format != "env" {
		return requestCommand{}, fmt.Errorf("format must be summary or env")
	}

	return requestCommand{
		Target:          target,
		DurationSeconds: int32(*duration),
		OutputFormat:    *format,
	}, nil
}

const maxInt32 = int64(1<<31 - 1)

func RunMain() {
	if err := Run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
