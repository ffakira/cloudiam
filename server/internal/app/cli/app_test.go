package cliapp

import "testing"

func TestParseRequestCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    requestCommand
		wantErr bool
	}{
		{
			name: "target request with duration",
			args: []string{"access", "request", "aws-sandbox-readonly", "--duration", "900"},
			want: requestCommand{
				Target:          "aws-sandbox-readonly",
				DurationSeconds: 900,
				OutputFormat:    "summary",
			},
		},
		{
			name: "target request with default duration",
			args: []string{"access", "request", "aws-sandbox-readonly"},
			want: requestCommand{
				Target:       "aws-sandbox-readonly",
				OutputFormat: "summary",
			},
		},
		{
			name: "target request with env format",
			args: []string{"access", "request", "aws-sandbox-readonly", "--format", "env"},
			want: requestCommand{
				Target:       "aws-sandbox-readonly",
				OutputFormat: "env",
			},
		},
		{
			name:    "json format is rejected",
			args:    []string{"access", "request", "aws-sandbox-readonly", "--format", "json"},
			wantErr: true,
		},
		{
			name:    "old provider flag is rejected",
			args:    []string{"access", "request", "aws-sandbox-readonly", "--provider", "aws"},
			wantErr: true,
		},
		{
			name:    "principal flag is rejected",
			args:    []string{"access", "request", "aws-sandbox-readonly", "--principal", "akira"},
			wantErr: true,
		},
		{
			name:    "address flag is rejected",
			args:    []string{"--addr", "127.0.0.1:50051", "access", "request", "aws-sandbox-readonly"},
			wantErr: true,
		},
		{
			name:    "missing target is rejected",
			args:    []string{"access", "request"},
			wantErr: true,
		},
		{
			name:    "misplaced duration flag is rejected as missing target",
			args:    []string{"access", "request", "--duration", "900"},
			wantErr: true,
		},
		{
			name:    "extra positional argument is rejected",
			args:    []string{"access", "request", "aws-sandbox-readonly", "extra-target"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRequestCommand(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse request command: %v", err)
			}
			if got != tt.want {
				t.Fatalf("command = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseCommandLogin(t *testing.T) {
	got, err := parseCommand([]string{"login"})
	if err != nil {
		t.Fatalf("parse login command: %v", err)
	}
	if got.Name != "login" {
		t.Fatalf("command name = %q, want login", got.Name)
	}
}

func TestParseCommandLoginRejectsAddressFlag(t *testing.T) {
	_, err := parseCommand([]string{"--addr", "127.0.0.1:50051", "login"})
	if err == nil {
		t.Fatal("expected error")
	}
}
