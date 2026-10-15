package config

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name: "valid config",
		},
		{
			name: "valid approval required target",
			mutate: func(cfg *Config) {
				target := cfg.Access.Targets["aws-sandbox-readonly"]
				target.ApprovalRequired = true
				cfg.Access.Targets["aws-sandbox-readonly"] = target
			},
		},
		{
			name: "unknown account reference",
			mutate: func(cfg *Config) {
				cfg.Access.Targets["aws-sandbox-readonly"] = TargetConfig{
					Provider:        "aws",
					Account:         "missing",
					Role:            "readonly",
					DefaultDuration: "15m",
					MaxDuration:     "1h",
				}
			},
			wantErr: true,
		},
		{
			name: "unknown role reference",
			mutate: func(cfg *Config) {
				target := cfg.Access.Targets["aws-sandbox-readonly"]
				target.Role = "missing"
				cfg.Access.Targets["aws-sandbox-readonly"] = target
			},
			wantErr: true,
		},
		{
			name: "unknown group reference",
			mutate: func(cfg *Config) {
				cfg.Access.Policies[0].Groups = []string{"missing"}
			},
			wantErr: true,
		},
		{
			name: "unknown target reference",
			mutate: func(cfg *Config) {
				cfg.Access.Policies[0].Targets = []string{"missing"}
			},
			wantErr: true,
		},
		{
			name: "invalid duration",
			mutate: func(cfg *Config) {
				target := cfg.Access.Targets["aws-sandbox-readonly"]
				target.DefaultDuration = "soon"
				cfg.Access.Targets["aws-sandbox-readonly"] = target
			},
			wantErr: true,
		},
		{
			name: "default duration exceeds max duration",
			mutate: func(cfg *Config) {
				target := cfg.Access.Targets["aws-sandbox-readonly"]
				target.DefaultDuration = "2h"
				target.MaxDuration = "1h"
				cfg.Access.Targets["aws-sandbox-readonly"] = target
			},
			wantErr: true,
		},
		{
			name: "invalid aws account id",
			mutate: func(cfg *Config) {
				account := cfg.Providers.AWS.Accounts["sandbox"]
				account.ID = "<AWS_ACCOUNT_ID>"
				cfg.Providers.AWS.Accounts["sandbox"] = account
			},
			wantErr: true,
		},
		{
			name: "invalid aws role arn",
			mutate: func(cfg *Config) {
				account := cfg.Providers.AWS.Accounts["sandbox"]
				role := account.Roles["readonly"]
				role.ARN = "arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly"
				account.Roles["readonly"] = role
				cfg.Providers.AWS.Accounts["sandbox"] = account
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			err := Validate(cfg)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func validConfig() Config {
	return Config{
		Version: CurrentVersion,
		Tenant:  "<COMPANY_UUID>",
		Serve: ServeConfig{
			GRPC: GRPCConfig{Addr: "127.0.0.1:50051"},
		},
		Providers: ProvidersConfig{
			AWS: AWSProviderConfig{
				Region: "us-east-1",
				Accounts: map[string]AccountConfig{
					"sandbox": {
						ID: "000000000000",
						Roles: map[string]RoleConfig{
							"readonly": {ARN: "arn:aws:iam::000000000000:role/jit/readonly"},
						},
					},
				},
			},
		},
		Principals: PrincipalsConfig{
			Users: map[string]UserConfig{
				"akira": {Groups: []string{"developers"}},
			},
		},
		Access: AccessConfig{
			Targets: map[string]TargetConfig{
				"aws-sandbox-readonly": {
					Provider:        "aws",
					Account:         "sandbox",
					Role:            "readonly",
					DefaultDuration: "15m",
					MaxDuration:     "1h",
				},
			},
			Policies: []PolicyConfig{
				{
					Name:    "developers-sandbox-readonly",
					Groups:  []string{"developers"},
					Targets: []string{"aws-sandbox-readonly"},
				},
			},
		},
	}
}
