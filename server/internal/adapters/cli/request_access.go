package cliadapter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RequestAccessInput struct {
	CloudIAMSessionToken string
	Target               string
	DurationSeconds      int32
	OutputFormat         string
}

func RequestAccess(ctx context.Context, client accessv1.AccessServiceClient, input RequestAccessInput, out io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := client.RequestAccess(ctx, &accessv1.RequestAccessRequest{
		CloudiamSessionToken: input.CloudIAMSessionToken,
		Target:               input.Target,
		DurationSeconds:      input.DurationSeconds,
	})
	if err != nil {
		if friendly := friendlyRequestAccessError(input.Target, err); friendly != "" {
			return fmt.Errorf("%s", friendly)
		}
		return fmt.Errorf("request access: %w", err)
	}

	switch outputFormat(input.OutputFormat) {
	case "summary":
		if resp.GetStatus() == "pending_approval" {
			return writePendingApprovalOutput(out, input, resp)
		}
		return writeSummaryOutput(out, input, resp)
	case "env":
		if resp.GetStatus() == "pending_approval" {
			return fmt.Errorf("credentials are pending approval and cannot be printed as env")
		}
		return writeEnvOutput(out, resp)
	default:
		return fmt.Errorf("unsupported output format %q", input.OutputFormat)
	}
}

func friendlyRequestAccessError(target string, err error) string {
	statusErr, ok := status.FromError(err)
	if !ok {
		return ""
	}
	switch statusErr.Code() {
	case codes.FailedPrecondition:
		if statusErr.Message() == "credential issuance denied by cloud provider" {
			return fmt.Sprintf(
				"access request failed for target %q: AWS denied credential issuance. Ask an operator to verify the target role trust policy allows the CloudIAM broker to use sts:AssumeRole, sts:SetSourceIdentity, and sts:TagSession.",
				target,
			)
		}
	case codes.PermissionDenied:
		return fmt.Sprintf("access denied for target %q", target)
	case codes.Unauthenticated:
		return "cloudiam session is invalid or expired; run login again"
	}
	return ""
}

func outputFormat(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "summary"
	}
	return value
}

func writePendingApprovalOutput(out io.Writer, input RequestAccessInput, resp *accessv1.RequestAccessResponse) error {
	lines := []string{
		"access pending approval\n",
		fmt.Sprintf("target: %s\n", input.Target),
		fmt.Sprintf("access_request_id: %d\n", resp.GetAccessRequestId()),
		fmt.Sprintf("approval_expires_at: %s\n", resp.GetApprovalExpiresAt()),
		"credentials: not issued\n",
	}
	return writeLines(out, lines)
}

func writeSummaryOutput(out io.Writer, input RequestAccessInput, resp *accessv1.RequestAccessResponse) error {
	lines := []string{
		"access granted\n",
		fmt.Sprintf("target: %s\n", input.Target),
		fmt.Sprintf("expires_at: %s\n", resp.GetExpiresAt()),
		"credentials: hidden (rerun with --format env to print shell exports)\n",
	}
	return writeLines(out, lines)
}

func writeEnvOutput(out io.Writer, resp *accessv1.RequestAccessResponse) error {
	lines := []string{
		fmt.Sprintf("export AWS_ACCESS_KEY_ID=%q\n", resp.GetAccessKeyId()),
		fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=%q\n", resp.GetSecretAccessKey()),
		fmt.Sprintf("export AWS_SESSION_TOKEN=%q\n", resp.GetSessionToken()),
		fmt.Sprintf("export AWS_SESSION_EXPIRATION=%q\n", resp.GetExpiresAt()),
	}
	return writeLines(out, lines)
}

func writeLines(out io.Writer, lines []string) error {
	for _, line := range lines {
		if _, err := io.WriteString(out, line); err != nil {
			return fmt.Errorf("write access output: %w", err)
		}
	}
	return nil
}
