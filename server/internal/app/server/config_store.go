package serverapp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	yamlconfig "github.com/ffakira/cloudiam/server/internal/adapters/config/yaml"
	"github.com/ffakira/cloudiam/server/internal/core/access"
	coreconfig "github.com/ffakira/cloudiam/server/internal/core/config"
	"github.com/ffakira/cloudiam/server/internal/core/policy"
)

type configSnapshot struct {
	generation uint64
	config     coreconfig.Config
	evaluator  *policy.Evaluator
}

type reloadStatus struct {
	Success    bool
	ConfigPath string
	Generation uint64
	Err        error
}

type configStore struct {
	path     string
	baseline restartFields
	current  atomic.Pointer[configSnapshot]
	reloadMu sync.Mutex
}

type restartFields struct {
	grpcAddr  string
	awsRegion string
}

func newConfigStore(path string, initial coreconfig.Config) *configStore {
	store := &configStore{
		path:     path,
		baseline: restartFieldsFromConfig(initial),
	}
	store.current.Store(newConfigSnapshot(1, initial))
	return store
}

func (s *configStore) Current() *configSnapshot {
	return s.current.Load()
}

func (s *configStore) Reload(ctx context.Context) reloadStatus {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	active := s.Current()
	if err := ctx.Err(); err != nil {
		return reloadStatus{Success: false, ConfigPath: s.path, Generation: active.generation, Err: err}
	}

	candidate, err := yamlconfig.LoadFile(s.path)
	if err != nil {
		return reloadStatus{Success: false, ConfigPath: s.path, Generation: active.generation, Err: err}
	}
	if err := s.baseline.validate(candidate); err != nil {
		return reloadStatus{Success: false, ConfigPath: s.path, Generation: active.generation, Err: err}
	}

	next := newConfigSnapshot(active.generation+1, candidate)
	s.current.Store(next)
	return reloadStatus{Success: true, ConfigPath: s.path, Generation: next.generation}
}

func newConfigSnapshot(generation uint64, cfg coreconfig.Config) *configSnapshot {
	cloned := cloneConfig(cfg)
	return &configSnapshot{
		generation: generation,
		config:     cloned,
		evaluator:  policy.NewEvaluator(cloned),
	}
}

func restartFieldsFromConfig(cfg coreconfig.Config) restartFields {
	return restartFields{
		grpcAddr:  cfg.Serve.GRPC.Addr,
		awsRegion: cfg.Providers.AWS.Region,
	}
}

func (f restartFields) validate(candidate coreconfig.Config) error {
	candidateFields := restartFieldsFromConfig(candidate)
	if candidateFields.grpcAddr != f.grpcAddr {
		return fmt.Errorf("serve.grpc.addr changed from %q to %q and requires server restart", f.grpcAddr, candidateFields.grpcAddr)
	}
	if candidateFields.awsRegion != f.awsRegion {
		return fmt.Errorf("providers.aws.region changed from %q to %q and requires server restart", f.awsRegion, candidateFields.awsRegion)
	}
	return nil
}

type snapshotRequestEvaluator struct {
	store *configStore
}

func (e snapshotRequestEvaluator) EvaluateRequest(principal, target string, requestedDuration time.Duration) (access.AuthorizationDecision, access.ResolvedTarget, error) {
	snapshot := e.store.Current()
	decision := snapshot.evaluator.Evaluate(principal, target)
	if !decision.Allowed {
		return access.AuthorizationDecision{
			Allowed: false,
			Reason:  decision.Reason,
		}, access.ResolvedTarget{}, nil
	}

	resolved, err := snapshot.config.ResolveTarget(target, requestedDuration)
	if err != nil {
		return access.AuthorizationDecision{}, access.ResolvedTarget{}, err
	}
	return access.AuthorizationDecision{
			Allowed:          true,
			Reason:           decision.Reason,
			ApprovalRequired: decision.ApprovalRequired,
		}, access.ResolvedTarget{
			Name:     resolved.Name,
			Provider: resolved.Provider,
			Account:  resolved.Account,
			Role:     resolved.Role,
			RoleARN:  resolved.RoleARN,
			Duration: resolved.Duration,
		}, nil
}

func cloneConfig(cfg coreconfig.Config) coreconfig.Config {
	cfg.Providers.AWS.Accounts = cloneAccounts(cfg.Providers.AWS.Accounts)
	cfg.Principals.Users = cloneUsers(cfg.Principals.Users)
	cfg.Access.Targets = cloneTargets(cfg.Access.Targets)
	cfg.Access.Policies = clonePolicies(cfg.Access.Policies)
	return cfg
}

func cloneAccounts(accounts map[string]coreconfig.AccountConfig) map[string]coreconfig.AccountConfig {
	if accounts == nil {
		return nil
	}
	cloned := make(map[string]coreconfig.AccountConfig, len(accounts))
	for name, account := range accounts {
		account.Roles = cloneRoles(account.Roles)
		cloned[name] = account
	}
	return cloned
}

func cloneRoles(roles map[string]coreconfig.RoleConfig) map[string]coreconfig.RoleConfig {
	if roles == nil {
		return nil
	}
	cloned := make(map[string]coreconfig.RoleConfig, len(roles))
	for name, role := range roles {
		cloned[name] = role
	}
	return cloned
}

func cloneUsers(users map[string]coreconfig.UserConfig) map[string]coreconfig.UserConfig {
	if users == nil {
		return nil
	}
	cloned := make(map[string]coreconfig.UserConfig, len(users))
	for name, user := range users {
		user.Groups = append([]string(nil), user.Groups...)
		cloned[name] = user
	}
	return cloned
}

func cloneTargets(targets map[string]coreconfig.TargetConfig) map[string]coreconfig.TargetConfig {
	if targets == nil {
		return nil
	}
	cloned := make(map[string]coreconfig.TargetConfig, len(targets))
	for name, target := range targets {
		cloned[name] = target
	}
	return cloned
}

func clonePolicies(policies []coreconfig.PolicyConfig) []coreconfig.PolicyConfig {
	if policies == nil {
		return nil
	}
	cloned := make([]coreconfig.PolicyConfig, len(policies))
	for i, policy := range policies {
		policy.Groups = append([]string(nil), policy.Groups...)
		policy.Targets = append([]string(nil), policy.Targets...)
		cloned[i] = policy
	}
	return cloned
}
