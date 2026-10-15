package config

import (
	"fmt"
	"time"
)

type ResolvedTarget struct {
	Name     string
	Provider string
	Account  string
	Role     string
	RoleARN  string
	Duration time.Duration
}

func (cfg Config) ResolveTarget(targetName string, requestedDuration time.Duration) (ResolvedTarget, error) {
	target, ok := cfg.Access.Targets[targetName]
	if !ok {
		return ResolvedTarget{}, fmt.Errorf("target %q not found", targetName)
	}
	if requestedDuration < 0 {
		return ResolvedTarget{}, fmt.Errorf("requested duration must be positive")
	}

	defaultDuration, err := parsePositiveDuration(target.DefaultDuration, fmt.Sprintf("access.targets.%s.default_duration", targetName))
	if err != nil {
		return ResolvedTarget{}, err
	}
	maxDuration, err := parsePositiveDuration(target.MaxDuration, fmt.Sprintf("access.targets.%s.max_duration", targetName))
	if err != nil {
		return ResolvedTarget{}, err
	}

	duration := requestedDuration
	if duration == 0 {
		duration = defaultDuration
	}
	if duration > maxDuration {
		return ResolvedTarget{}, fmt.Errorf("requested duration exceeds target max duration")
	}

	account, ok := cfg.Providers.AWS.Accounts[target.Account]
	if !ok {
		return ResolvedTarget{}, fmt.Errorf("target %q references unknown account %q", targetName, target.Account)
	}
	role, ok := account.Roles[target.Role]
	if !ok {
		return ResolvedTarget{}, fmt.Errorf("target %q references unknown role %q", targetName, target.Role)
	}

	return ResolvedTarget{
		Name:     targetName,
		Provider: target.Provider,
		Account:  target.Account,
		Role:     target.Role,
		RoleARN:  role.ARN,
		Duration: duration,
	}, nil
}
