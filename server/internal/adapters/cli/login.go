package cliadapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	authv1 "github.com/ffakira/cloudiam/server/gen/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginInput struct {
	PollTimeout time.Duration
}

type AuthClient interface {
	StartDeviceLogin(ctx context.Context, in *authv1.StartDeviceLoginRequest, opts ...grpc.CallOption) (*authv1.StartDeviceLoginResponse, error)
	ExchangeDeviceCode(ctx context.Context, in *authv1.ExchangeDeviceCodeRequest, opts ...grpc.CallOption) (*authv1.ExchangeDeviceCodeResponse, error)
}

type SessionSaver interface {
	Save(session Session) error
}

func Login(ctx context.Context, client AuthClient, store SessionSaver, input LoginInput, out io.Writer) error {
	if input.PollTimeout <= 0 {
		input.PollTimeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, input.PollTimeout)
	defer cancel()

	start, err := client.StartDeviceLogin(ctx, &authv1.StartDeviceLoginRequest{})
	if err != nil {
		return fmt.Errorf("start device login: %w", err)
	}

	if _, err := fmt.Fprintf(out, "Open: %s\n", start.GetVerificationUri()); err != nil {
		return fmt.Errorf("write verification uri: %w", err)
	}
	if _, err := fmt.Fprintf(out, "Code: %s\n", start.GetUserCode()); err != nil {
		return fmt.Errorf("write user code: %w", err)
	}

	interval := time.Duration(start.GetIntervalSeconds()) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("login timed out: %w", ctx.Err())
		case <-timer.C:
		}

		session, err := client.ExchangeDeviceCode(ctx, &authv1.ExchangeDeviceCodeRequest{
			DeviceCode: start.GetDeviceCode(),
		})
		if err == nil {
			expiresAt, parseErr := time.Parse(time.RFC3339, session.GetExpiresAt())
			if parseErr != nil {
				return fmt.Errorf("parse session expiry: %w", parseErr)
			}
			if err := store.Save(Session{
				Version:     "v1",
				Workspace:   session.GetWorkspace(),
				Principal:   session.GetPrincipal(),
				Endpoint:    session.GetEndpoint(),
				AccessToken: session.GetAccessToken(),
				ExpiresAt:   expiresAt,
			}); err != nil {
				return fmt.Errorf("save session: %w", err)
			}
			if _, err := fmt.Fprintf(out, "Logged in as %s\n", session.GetPrincipal()); err != nil {
				return fmt.Errorf("write login result: %w", err)
			}
			return nil
		}

		code := status.Code(err)
		switch code {
		case codes.FailedPrecondition:
			timer.Reset(interval)
			continue
		case codes.PermissionDenied, codes.DeadlineExceeded, codes.NotFound:
			return fmt.Errorf("login failed: %w", err)
		default:
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("login timed out: %w", ctx.Err())
			}
			return fmt.Errorf("exchange device code: %w", err)
		}
	}
}
