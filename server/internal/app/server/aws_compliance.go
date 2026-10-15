package serverapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	yamlconfig "github.com/ffakira/cloudiam/server/internal/adapters/config/yaml"
	"github.com/ffakira/cloudiam/server/internal/core/compliance"
)

const defaultBrokerUserName = "cloud-iam-broker"

var ErrComplianceFailed = errors.New("aws policy compliance check failed")

type TrustPolicyReader interface {
	ReadTrustPolicy(ctx context.Context, roleARN string) ([]byte, error)
}

type ScanAWSComplianceInput struct {
	ConfigPath     string
	BrokerUserName string
}

func ScanAWSCompliance(ctx context.Context, reader TrustPolicyReader, input ScanAWSComplianceInput, out io.Writer) error {
	if strings.TrimSpace(input.ConfigPath) == "" {
		return fmt.Errorf("config path is required")
	}
	if reader == nil {
		return fmt.Errorf("trust policy reader is required")
	}
	if strings.TrimSpace(input.BrokerUserName) == "" {
		input.BrokerUserName = defaultBrokerUserName
	}

	cfg, err := yamlconfig.LoadFile(input.ConfigPath)
	if err != nil {
		return err
	}

	targetNames := make([]string, 0, len(cfg.Access.Targets))
	for targetName := range cfg.Access.Targets {
		targetNames = append(targetNames, targetName)
	}
	sort.Strings(targetNames)

	var failed bool
	for _, targetName := range targetNames {
		resolved, err := cfg.ResolveTarget(targetName, 0)
		if err != nil {
			return fmt.Errorf("resolve target %q: %w", targetName, err)
		}
		if resolved.Provider != "aws" {
			continue
		}

		account := cfg.Providers.AWS.Accounts[resolved.Account]
		brokerARN := fmt.Sprintf("arn:aws:iam::%s:user/%s", account.ID, input.BrokerUserName)
		policyJSON, err := reader.ReadTrustPolicy(ctx, resolved.RoleARN)
		if err != nil {
			return fmt.Errorf("read trust policy for target %q: %w", targetName, err)
		}

		result, err := compliance.ScanTrustPolicy(compliance.TargetInput{
			TargetName:         targetName,
			RoleARN:            resolved.RoleARN,
			BrokerPrincipalARN: brokerARN,
			TrustPolicyJSON:    policyJSON,
		})
		if err != nil {
			return fmt.Errorf("scan trust policy for target %q: %w", targetName, err)
		}
		if err := writeComplianceResult(out, result); err != nil {
			return err
		}
		if !result.Compliant() {
			failed = true
		}
	}

	if failed {
		return ErrComplianceFailed
	}
	return nil
}

func writeComplianceResult(out io.Writer, result compliance.Result) error {
	status := "compliant"
	if !result.Compliant() {
		status = "non-compliant"
	}

	if _, err := fmt.Fprintf(out, "target: %s\n", result.TargetName); err != nil {
		return fmt.Errorf("write compliance target: %w", err)
	}
	if _, err := fmt.Fprintf(out, "role: %s\n", maskAWSAccountID(result.RoleARN)); err != nil {
		return fmt.Errorf("write compliance role: %w", err)
	}
	if _, err := fmt.Fprintf(out, "status: %s\n", status); err != nil {
		return fmt.Errorf("write compliance status: %w", err)
	}
	if result.Compliant() {
		if _, err := fmt.Fprintln(out); err != nil {
			return fmt.Errorf("write compliance spacing: %w", err)
		}
		return nil
	}

	if _, err := fmt.Fprintln(out, "findings:"); err != nil {
		return fmt.Errorf("write compliance findings header: %w", err)
	}
	for _, finding := range result.Findings {
		if _, err := fmt.Fprintf(out, "  - %s: %s\n", finding.ID, finding.Message); err != nil {
			return fmt.Errorf("write compliance finding: %w", err)
		}
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return fmt.Errorf("write compliance spacing: %w", err)
	}
	return nil
}

var awsAccountIDInARNPattern = regexp.MustCompile(`arn:aws([a-zA-Z-]*):iam::[0-9]{12}:`)

func maskAWSAccountID(value string) string {
	return awsAccountIDInARNPattern.ReplaceAllString(value, "arn:aws$1:iam::<AWS_ACCOUNT_ID>:")
}
