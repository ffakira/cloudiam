package yamlconfig

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	coreconfig "github.com/ffakira/cloudiam/server/internal/core/config"
	"gopkg.in/yaml.v3"
)

var secretKeys = map[string]struct{}{
	"access_key":            {},
	"access_key_id":         {},
	"secret_access_key":     {},
	"session_token":         {},
	"aws_access_key_id":     {},
	"aws_secret_access_key": {},
	"aws_session_token":     {},
	"cloudiam_token":        {},
	"token":                 {},
	"password":              {},
	"secret":                {},
}

func LoadFile(path string) (coreconfig.Config, error) {
	// #nosec G304 -- config path is an operator-provided local file path, not request-controlled input.
	file, err := os.Open(path)
	if err != nil {
		return coreconfig.Config{}, fmt.Errorf("open config: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	cfg, err := Load(file)
	if err != nil {
		return coreconfig.Config{}, fmt.Errorf("load config %q: %w", path, err)
	}
	return cfg, nil
}

func Load(r io.Reader) (coreconfig.Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return coreconfig.Config{}, fmt.Errorf("read config: %w", err)
	}
	data = []byte(os.ExpandEnv(string(data)))

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return coreconfig.Config{}, fmt.Errorf("parse yaml: %w", err)
	}
	if err := rejectSecretKeys(&node); err != nil {
		return coreconfig.Config{}, err
	}

	var cfg coreconfig.Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return coreconfig.Config{}, fmt.Errorf("decode yaml: %w", err)
	}
	if err := coreconfig.Validate(cfg); err != nil {
		return coreconfig.Config{}, err
	}
	return cfg, nil
}

func rejectSecretKeys(node *yaml.Node) error {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if _, ok := secretKeys[normalizeKey(key.Value)]; ok {
				return fmt.Errorf("config contains forbidden secret field %q", key.Value)
			}
			if err := rejectSecretKeys(node.Content[i+1]); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := rejectSecretKeys(child); err != nil {
			return err
		}
	}
	return nil
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, "-", "_")
	return key
}
