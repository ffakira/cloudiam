package policy

import coreconfig "github.com/ffakira/cloudiam/server/internal/core/config"

type Decision struct {
	Allowed          bool
	Reason           string
	ApprovalRequired bool
}

type Evaluator struct {
	cfg coreconfig.Config
}

func NewEvaluator(cfg coreconfig.Config) *Evaluator {
	return &Evaluator{cfg: cfg}
}

func (e *Evaluator) Evaluate(principal, target string) Decision {
	user, ok := e.cfg.Principals.Users[principal]
	if !ok {
		return Decision{Allowed: false, Reason: "principal not found"}
	}
	targetConfig, ok := e.cfg.Access.Targets[target]
	if !ok {
		return Decision{Allowed: false, Reason: "target not found"}
	}

	userGroups := make(map[string]struct{}, len(user.Groups))
	for _, group := range user.Groups {
		userGroups[group] = struct{}{}
	}

	for _, policy := range e.cfg.Access.Policies {
		if !policyTargets(policy, target) {
			continue
		}
		for _, group := range policy.Groups {
			if _, ok := userGroups[group]; ok {
				reason := "policy " + policy.Name + " matched"
				if targetConfig.ApprovalRequired {
					reason += "; approval required"
				}
				return Decision{
					Allowed:          true,
					Reason:           reason,
					ApprovalRequired: targetConfig.ApprovalRequired,
				}
			}
		}
	}

	return Decision{Allowed: false, Reason: "no matching policy"}
}

func (e *Evaluator) Authorize(principal, target string) Decision {
	return e.Evaluate(principal, target)
}

func policyTargets(policy coreconfig.PolicyConfig, target string) bool {
	for _, policyTarget := range policy.Targets {
		if policyTarget == target {
			return true
		}
	}
	return false
}
