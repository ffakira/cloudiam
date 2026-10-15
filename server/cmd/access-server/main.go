package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	awsadapter "github.com/ffakira/cloudiam/server/internal/adapters/aws"
	yamlconfig "github.com/ffakira/cloudiam/server/internal/adapters/config/yaml"
	serverapp "github.com/ffakira/cloudiam/server/internal/app/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "validate-config" {
		validateConfig(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		healthcheck(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "check-aws-config" {
		checkAWSConfig(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "scan-aws-policies" {
		scanAWSPolicies(os.Args[2:])
		return
	}

	addr := flag.String("addr", "127.0.0.1:50051", "gRPC server address")
	configPath := flag.String("config", "", "CloudIAM YAML config path")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := serverapp.Run(ctx, serverapp.Config{Addr: *addr, ConfigPath: *configPath}); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanAWSPolicies(args []string) {
	fs := flag.NewFlagSet("scan-aws-policies", flag.ExitOnError)
	configPath := fs.String("config", "", "CloudIAM YAML config path")
	brokerUser := fs.String("broker-user", "cloud-iam-broker", "CloudIAM broker IAM user name")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "config path is required")
		os.Exit(1)
	}

	appCfg, err := yamlconfig.LoadFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(appCfg.Providers.AWS.Region))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	reader := awsadapter.NewRolePolicyReader(iam.NewFromConfig(awsCfg))
	err = serverapp.ScanAWSCompliance(ctx, reader, serverapp.ScanAWSComplianceInput{
		ConfigPath:     *configPath,
		BrokerUserName: *brokerUser,
	}, os.Stdout)
	if err != nil {
		if errors.Is(err, serverapp.ErrComplianceFailed) {
			fmt.Fprintln(os.Stderr, serverapp.ErrComplianceFailed)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateConfig(args []string) {
	fs := flag.NewFlagSet("validate-config", flag.ExitOnError)
	configPath := fs.String("config", "", "CloudIAM YAML config path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := serverapp.ValidateConfig(*configPath, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func healthcheck(args []string) {
	fs := flag.NewFlagSet("healthcheck", flag.ExitOnError)
	url := fs.String("url", "http://127.0.0.1:8080/healthz", "health check URL")
	timeout := fs.Duration("timeout", 3*time.Second, "health check timeout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Get(*url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "healthcheck failed: status %d\n", resp.StatusCode)
		os.Exit(1)
	}
}

func checkAWSConfig(args []string) {
	fs := flag.NewFlagSet("check-aws-config", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, path := range []string{os.Getenv("AWS_CONFIG_FILE"), os.Getenv("AWS_SHARED_CREDENTIALS_FILE")} {
		if path == "" {
			continue
		}
		if err := printProfileHeaders(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(1)
		}
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := fmt.Fprintf(os.Stdout, "aws config loaded for region %q\n", cfg.Region); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printProfileHeaders(path string) error {
	// #nosec G304 G703 -- diagnostic command intentionally opens operator-selected local AWS config files.
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) >= 2 && line[0] == '[' && line[len(line)-1] == ']' {
			if _, err := fmt.Fprintf(os.Stdout, "%s: %s\n", path, line); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
