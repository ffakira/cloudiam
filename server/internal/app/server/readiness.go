package serverapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type readinessDependency interface {
	Check(ctx context.Context) error
}

type readinessDependencyFunc func(ctx context.Context) error

func (f readinessDependencyFunc) Check(ctx context.Context) error {
	return f(ctx)
}

type readinessChecker struct {
	dependencies map[string]readinessDependency
	timeout      time.Duration
}

type readinessResponse struct {
	Status string           `json:"status"`
	Checks []readinessCheck `json:"checks"`
}

type readinessCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func newReadinessChecker(dependencies map[string]readinessDependency) readinessChecker {
	return readinessChecker{
		dependencies: dependencies,
		timeout:      2 * time.Second,
	}
}

func (c readinessChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		if c.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.timeout)
			defer cancel()
		}

		response := c.Check(ctx)
		statusCode := http.StatusOK
		if response.Status != "ready" {
			statusCode = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			return
		}
	}
}

func (c readinessChecker) Check(ctx context.Context) readinessResponse {
	names := make([]string, 0, len(c.dependencies))
	for name := range c.dependencies {
		names = append(names, name)
	}
	sort.Strings(names)

	response := readinessResponse{
		Status: "ready",
		Checks: make([]readinessCheck, 0, len(names)),
	}
	for _, name := range names {
		dependency := c.dependencies[name]
		check := readinessCheck{Name: name, Status: "ok"}
		if dependency == nil {
			check.Status = "failed"
			check.Error = "readiness dependency is not configured"
		} else if err := dependency.Check(ctx); err != nil {
			check.Status = "failed"
			check.Error = safeErrorMessage(err)
		}
		if check.Status != "ok" {
			response.Status = "not_ready"
		}
		response.Checks = append(response.Checks, check)
	}
	return response
}

func configReadinessDependency(store *configStore) readinessDependency {
	return readinessDependencyFunc(func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if store == nil || store.Current() == nil {
			return fmt.Errorf("active config snapshot is unavailable")
		}
		return nil
	})
}

type pingReadinessDependency struct {
	name   string
	pinger interface {
		PingContext(context.Context) error
	}
}

func (d pingReadinessDependency) Check(ctx context.Context) error {
	if d.pinger == nil {
		return fmt.Errorf("%s dependency is not configured", d.name)
	}
	if err := d.pinger.PingContext(ctx); err != nil {
		return fmt.Errorf("%s ping failed: %w", d.name, err)
	}
	return nil
}
