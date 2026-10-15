package config

import (
	"testing"
	"time"
)

func TestConfig_ResolveTarget(t *testing.T) {
	tests := []struct {
		name              string
		targetName        string
		requestedDuration time.Duration
		wantDuration      time.Duration
		wantErr           bool
	}{
		{
			name:         "valid target uses default duration",
			targetName:   "aws-sandbox-readonly",
			wantDuration: 15 * time.Minute,
		},
		{
			name:              "valid target accepts duration override",
			targetName:        "aws-sandbox-readonly",
			requestedDuration: 30 * time.Minute,
			wantDuration:      30 * time.Minute,
		},
		{
			name:              "duration above max fails",
			targetName:        "aws-sandbox-readonly",
			requestedDuration: 2 * time.Hour,
			wantErr:           true,
		},
		{
			name:       "unknown target fails",
			targetName: "missing",
			wantErr:    true,
		},
		{
			name:              "negative duration fails",
			targetName:        "aws-sandbox-readonly",
			requestedDuration: -time.Second,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := validConfig().ResolveTarget(tt.targetName, tt.requestedDuration)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.Name != tt.targetName {
				t.Fatalf("resolved name = %q, want %q", resolved.Name, tt.targetName)
			}
			if resolved.Provider != "aws" {
				t.Fatalf("provider = %q, want aws", resolved.Provider)
			}
			if resolved.Account != "sandbox" {
				t.Fatalf("account = %q, want sandbox", resolved.Account)
			}
			if resolved.Role != "readonly" {
				t.Fatalf("role = %q, want readonly", resolved.Role)
			}
			if resolved.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
				t.Fatalf("role arn = %q", resolved.RoleARN)
			}
			if resolved.Duration != tt.wantDuration {
				t.Fatalf("duration = %s, want %s", resolved.Duration, tt.wantDuration)
			}
		})
	}
}
