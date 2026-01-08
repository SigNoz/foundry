// Package main provides the foundryctl CLI tool for managing deployments.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cuelang.org/go/cue/cuecontext"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/loader"
	"github.com/spf13/cobra"
)

type commands []string

type castOptions struct {
	Platform string
	Execute  commands
}

func registerCastCmd(rootCmd *cobra.Command) {
	castCmd := &cobra.Command{
		Use:   "cast",
		Short: "Cast to the target environment.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "cast"))
			ctx := cmd.Context()

			logger.DebugContext(ctx, "starting command", slog.String("cfg.file", cfg.File))
			cuectx := cuecontext.New()
			config, err := loader.LoadConfig(cuectx, cfg.File)
			if err != nil {
				logger.ErrorContext(ctx, "config load failed", slog.String("error", err.Error()))
				return err
			}
			platform := config.Platform
			logger.DebugContext(ctx, "Configuration loaded", slog.String("Platform detected:", platform))
			switch platform {
			case "docker":
				logger.InfoContext(ctx, "Docker platform selected", slog.String("Action", "Running docker compose"))
				// Run Docker compose as a subprocess
				dockerCast := castOptions{
					Platform: "docker",
					Execute:  commands{"cd ./pours/docker", "docker compose up -d"},
				}
				if err := runCommand(ctx, logger, dockerCast); err != nil {
					return err
				}
				if err := validateInstallation(ctx, logger, platform); err != nil {
					return err
				}

			case "linux":
				logger.InfoContext(ctx, "Linux platform selected", slog.String("Action", "Generating Linux deployment files"))
				// TBD
				if err := validateInstallation(ctx, logger, platform); err != nil {
					return err
				}
			}
			return nil
		},
	}
	rootCmd.AddCommand(castCmd)
}

// runCommand executes the commands specified in castOptions as a subprocess.
func runCommand(ctx context.Context, logger *slog.Logger, cast castOptions) error {
	logger.InfoContext(ctx, "Executing commands for platform", slog.String("platform", cast.Platform))

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Join commands with && to run in sequence
	command := strings.Join(cast.Execute, " && ")

	logger.DebugContext(runctx, "Running command", slog.String("command", command))

	cmd := exec.CommandContext(runctx, "sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		logger.ErrorContext(runctx, "Command execution failed", slog.String("error", err.Error()))
		return err
	}

	logger.InfoContext(runctx, "Command executed successfully")
	return nil
}

// validateInstallation ensures that the system is up and running depending on the platform.
func validateInstallation(ctx context.Context, logger *slog.Logger, platform string) error {
	logger.InfoContext(ctx, "Validating installation for platform", slog.String("platform", platform))

	switch platform {
	case "docker":
		// For docker, hit localhost:8080
		valctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Run curl to get HTTP status code
		cmd := exec.CommandContext(valctx, "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8080")
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = os.Stderr

		// Execute the command
		err := cmd.Run()
		if err != nil {
			logger.ErrorContext(valctx, "Validation failed: unable to reach localhost:8080", slog.String("error", err.Error()))
			return err
		}

		status := strings.TrimSpace(buf.String())
		code, parseErr := strconv.Atoi(status)
		if parseErr != nil {
			logger.ErrorContext(valctx, "Failed to parse HTTP status code", slog.String("status", status), slog.String("error", parseErr.Error()))
			return parseErr
		}

		logger.InfoContext(valctx, "Validation completed", slog.Int("http_code", code))

		if code == 200 {
			logger.InfoContext(valctx, "Validation successful: SigNoz is running on localhost:8080")
			logger.Info("Open your browser on http://localhost:8080")
			return nil
		}
		logger.ErrorContext(valctx, "Validation failed: unexpected HTTP status code", slog.Int("code", code))
		return fmt.Errorf("unexpected HTTP status code: %d", code)

	default:
		logger.InfoContext(ctx, "Validation not implemented for platform", slog.String("platform", platform))
		return nil
	}
}
