package config

type Config struct {
	Version    string           `yaml:"version"`
	Tenant     string           `yaml:"tenant"`
	Serve      ServeConfig      `yaml:"serve"`
	Providers  ProvidersConfig  `yaml:"providers"`
	Principals PrincipalsConfig `yaml:"principals"`
	Access     AccessConfig     `yaml:"access"`
}

type ServeConfig struct {
	GRPC GRPCConfig `yaml:"grpc"`
}

type GRPCConfig struct {
	Addr string `yaml:"addr"`
}

type ProvidersConfig struct {
	AWS AWSProviderConfig `yaml:"aws"`
}

type AWSProviderConfig struct {
	Region   string                   `yaml:"region"`
	Accounts map[string]AccountConfig `yaml:"accounts"`
}

type AccountConfig struct {
	ID    string                `yaml:"id"`
	Roles map[string]RoleConfig `yaml:"roles"`
}

type RoleConfig struct {
	ARN string `yaml:"arn"`
}

type PrincipalsConfig struct {
	Users map[string]UserConfig `yaml:"users"`
}

type UserConfig struct {
	Groups []string `yaml:"groups"`
}

type AccessConfig struct {
	Targets  map[string]TargetConfig `yaml:"targets"`
	Policies []PolicyConfig          `yaml:"policies"`
}

type TargetConfig struct {
	Provider         string `yaml:"provider"`
	Account          string `yaml:"account"`
	Role             string `yaml:"role"`
	DefaultDuration  string `yaml:"default_duration"`
	MaxDuration      string `yaml:"max_duration"`
	ApprovalRequired bool   `yaml:"approval_required"`
}

type PolicyConfig struct {
	Name    string   `yaml:"name"`
	Groups  []string `yaml:"groups"`
	Targets []string `yaml:"targets"`
}
