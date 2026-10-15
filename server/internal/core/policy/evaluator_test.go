package policy

import (
	"testing"

	coreconfig "github.com/ffakira/cloudiam/server/internal/core/config"
)

func TestEvaluator_Evaluate(t *testing.T) {
	tests := []struct {
		name                 string
		mutate               func(*coreconfig.Config)
		principal            string
		target               string
		wantAllow            bool
		wantApprovalRequired bool
	}{
		{
			name:      "matching group and target is allowed",
			principal: "akira",
			target:    "aws-sandbox-readonly",
			wantAllow: true,
		},
		{
			name: "matching approval required target is allowed with approval required",
			mutate: func(cfg *coreconfig.Config) {
				target := cfg.Access.Targets["aws-sandbox-readonly"]
				target.ApprovalRequired = true
				cfg.Access.Targets["aws-sandbox-readonly"] = target
			},
			principal:            "akira",
			target:               "aws-sandbox-readonly",
			wantAllow:            true,
			wantApprovalRequired: true,
		},
		{
			name:      "unknown principal is denied",
			principal: "unknown",
			target:    "aws-sandbox-readonly",
		},
		{
			name:      "unknown target is denied",
			principal: "akira",
			target:    "missing",
		},
		{
			name: "principal in wrong group is denied",
			mutate: func(cfg *coreconfig.Config) {
				cfg.Principals.Users["akira"] = coreconfig.UserConfig{Groups: []string{"auditors"}}
			},
			principal: "akira",
			target:    "aws-sandbox-readonly",
		},
		{
			name: "no matching policy is denied",
			mutate: func(cfg *coreconfig.Config) {
				cfg.Access.Policies = nil
			},
			principal: "akira",
			target:    "aws-sandbox-readonly",
		},
		{
			name: "policy for different target is denied",
			mutate: func(cfg *coreconfig.Config) {
				cfg.Access.Policies[0].Targets = []string{"other-target"}
			},
			principal: "akira",
			target:    "aws-sandbox-readonly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			decision := NewEvaluator(cfg).Evaluate(tt.principal, tt.target)
			if decision.Allowed != tt.wantAllow {
				t.Fatalf("allowed = %v, want %v, reason = %q", decision.Allowed, tt.wantAllow, decision.Reason)
			}
			if decision.ApprovalRequired != tt.wantApprovalRequired {
				t.Fatalf("approval required = %v, want %v", decision.ApprovalRequired, tt.wantApprovalRequired)
			}
			if decision.Reason == "" {
				t.Fatal("reason is required")
			}
		})
	}
}

func validConfig() coreconfig.Config {
	return coreconfig.Config{
		Version: coreconfig.CurrentVersion,
		Tenant:  "<COMPANY_UUID>",
		Serve: coreconfig.ServeConfig{
			GRPC: coreconfig.GRPCConfig{Addr: "127.0.0.1:50051"},
		},
		Providers: coreconfig.ProvidersConfig{
			AWS: coreconfig.AWSProviderConfig{
				Region: "us-east-1",
				Accounts: map[string]coreconfig.AccountConfig{
					"sandbox": {
						ID: "000000000000",
						Roles: map[string]coreconfig.RoleConfig{
							"readonly": {ARN: "arn:aws:iam::000000000000:role/jit/readonly"},
						},
					},
				},
			},
		},
		Principals: coreconfig.PrincipalsConfig{
			Users: map[string]coreconfig.UserConfig{
				"akira": {Groups: []string{"developers"}},
			},
		},
		Access: coreconfig.AccessConfig{
			Targets: map[string]coreconfig.TargetConfig{
				"aws-sandbox-readonly": {
					Provider:        "aws",
					Account:         "sandbox",
					Role:            "readonly",
					DefaultDuration: "15m",
					MaxDuration:     "1h",
				},
				"other-target": {
					Provider:        "aws",
					Account:         "sandbox",
					Role:            "readonly",
					DefaultDuration: "15m",
					MaxDuration:     "1h",
				},
			},
			Policies: []coreconfig.PolicyConfig{
				{
					Name:    "developers-sandbox-readonly",
					Groups:  []string{"developers"},
					Targets: []string{"aws-sandbox-readonly"},
				},
			},
		},
	}
}
