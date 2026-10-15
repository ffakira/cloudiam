package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const CurrentVersion = "v1"

var (
	awsAccountIDPattern = regexp.MustCompile(`^[0-9]{12}$`)
	awsRoleARNPattern   = regexp.MustCompile(`^arn:aws[a-zA-Z-]*:iam::[0-9]{12}:role/.+`)
)

func Validate(cfg Config) error {
	if strings.TrimSpace(cfg.Version) != CurrentVersion {
		return fmt.Errorf("version must be %q", CurrentVersion)
	}
	if strings.TrimSpace(cfg.Tenant) == "" {
		return fmt.Errorf("tenant is required")
	}
	if strings.TrimSpace(cfg.Serve.GRPC.Addr) == "" {
		return fmt.Errorf("serve.grpc.addr is required")
	}
	if strings.TrimSpace(cfg.Providers.AWS.Region) == "" {
		return fmt.Errorf("providers.aws.region is required")
	}
	if len(cfg.Providers.AWS.Accounts) == 0 {
		return fmt.Errorf("providers.aws.accounts is required")
	}
	if len(cfg.Access.Targets) == 0 {
		return fmt.Errorf("access.targets is required")
	}
	if len(cfg.Access.Policies) == 0 {
		return fmt.Errorf("access.policies is required")
	}

	groups := map[string]struct{}{}
	for userName, user := range cfg.Principals.Users {
		if strings.TrimSpace(userName) == "" {
			return fmt.Errorf("principals.users contains empty user name")
		}
		for _, group := range user.Groups {
			if strings.TrimSpace(group) == "" {
				return fmt.Errorf("principals.users.%s.groups contains empty group", userName)
			}
			groups[group] = struct{}{}
		}
	}

	for accountName, account := range cfg.Providers.AWS.Accounts {
		if strings.TrimSpace(accountName) == "" {
			return fmt.Errorf("providers.aws.accounts contains empty account name")
		}
		if strings.TrimSpace(account.ID) == "" {
			return fmt.Errorf("providers.aws.accounts.%s.id is required", accountName)
		}
		if !awsAccountIDPattern.MatchString(account.ID) {
			return fmt.Errorf("providers.aws.accounts.%s.id must be a 12 digit AWS account ID", accountName)
		}
		if len(account.Roles) == 0 {
			return fmt.Errorf("providers.aws.accounts.%s.roles is required", accountName)
		}
		for roleName, role := range account.Roles {
			if strings.TrimSpace(roleName) == "" {
				return fmt.Errorf("providers.aws.accounts.%s.roles contains empty role name", accountName)
			}
			if strings.TrimSpace(role.ARN) == "" {
				return fmt.Errorf("providers.aws.accounts.%s.roles.%s.arn is required", accountName, roleName)
			}
			if !awsRoleARNPattern.MatchString(role.ARN) {
				return fmt.Errorf("providers.aws.accounts.%s.roles.%s.arn must be a valid AWS IAM role ARN", accountName, roleName)
			}
		}
	}

	for targetName, target := range cfg.Access.Targets {
		if strings.TrimSpace(targetName) == "" {
			return fmt.Errorf("access.targets contains empty target name")
		}
		if strings.TrimSpace(target.Provider) != "aws" {
			return fmt.Errorf("access.targets.%s.provider references unknown provider %q", targetName, target.Provider)
		}
		account, ok := cfg.Providers.AWS.Accounts[target.Account]
		if !ok {
			return fmt.Errorf("access.targets.%s.account references unknown account %q", targetName, target.Account)
		}
		if _, ok := account.Roles[target.Role]; !ok {
			return fmt.Errorf("access.targets.%s.role references unknown role %q", targetName, target.Role)
		}
		defaultDuration, err := parsePositiveDuration(target.DefaultDuration, fmt.Sprintf("access.targets.%s.default_duration", targetName))
		if err != nil {
			return err
		}
		maxDuration, err := parsePositiveDuration(target.MaxDuration, fmt.Sprintf("access.targets.%s.max_duration", targetName))
		if err != nil {
			return err
		}
		if defaultDuration > maxDuration {
			return fmt.Errorf("access.targets.%s.default_duration exceeds max_duration", targetName)
		}
	}

	policyNames := map[string]struct{}{}
	for _, policy := range cfg.Access.Policies {
		if strings.TrimSpace(policy.Name) == "" {
			return fmt.Errorf("access.policies contains policy with empty name")
		}
		if _, exists := policyNames[policy.Name]; exists {
			return fmt.Errorf("access.policies contains duplicate policy name %q", policy.Name)
		}
		policyNames[policy.Name] = struct{}{}

		for _, group := range policy.Groups {
			if _, ok := groups[group]; !ok {
				return fmt.Errorf("access.policies.%s.groups references unknown group %q", policy.Name, group)
			}
		}
		for _, target := range policy.Targets {
			if _, ok := cfg.Access.Targets[target]; !ok {
				return fmt.Errorf("access.policies.%s.targets references unknown target %q", policy.Name, target)
			}
		}
	}

	return nil
}

func parsePositiveDuration(value, field string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s is invalid: %w", field, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", field)
	}
	return duration, nil
}
