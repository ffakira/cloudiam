package serverapp

import (
	"fmt"
	"io"
	"strings"

	yamlconfig "github.com/ffakira/cloudiam/server/internal/adapters/config/yaml"
)

func ValidateConfig(path string, out io.Writer) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is required")
	}
	if _, err := yamlconfig.LoadFile(path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "config %q is valid\n", path); err != nil {
		return fmt.Errorf("write config validation result: %w", err)
	}
	return nil
}
