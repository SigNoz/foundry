package main

import (
	"fmt"
	"log/slog"
	"os/exec"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/loader"
	"github.com/spf13/cobra"
)

// func RunGauge is the main function for the gauge command.
func runGauge(cmd *cobra.Command, _ []string) error {
	logger := instrumentation.NewLogger(cfg.Debug).With(slog.String("cmd.name", "gauge"))
	ctx := cmd.Context()
	cuectx := cuecontext.New()
	logger.DebugContext(ctx, "Starting Gauge command, using:", slog.String("cfg.file", cfg.File))
	config, err := loader.LoadConfig(cuectx, cfg.File)
	if err != nil {
		logger.ErrorContext(ctx, "failed to validate config", slog.String("cfg.file", cfg.File), foundryerrors.LogAttr(err))
		return err
	}
	requirements, err := getRequirements(config.Unified, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get requirements from config", slog.String("cfg.file", cfg.File), slog.String("error", err.Error()))
		return err
	}
	logger.InfoContext(ctx, "Required tools:", slog.String("tools", fmt.Sprintf("%v", requirements)))
	for _, v := range requirements {
		err := checkToolExists(v)
		if err != nil {
			logger.ErrorContext(ctx, "Tool not found", slog.String("tool.name", v))
			logger.Error(fmt.Sprintf("Unable to proceed, please install %v and try again.", v))
		} else {
			logger.InfoContext(ctx, "Tool is available", slog.String("tool.name", v))
		}
	}
	return nil
}

// func getRequirements extracts the list of required tools from the CUE configuration.
func getRequirements(cueFile cue.Value, logger *slog.Logger) ([]string, error) {
	var requirements []string
	reqList := cueFile.LookupPath(cue.ParsePath("requirements"))
	logger.Debug("Looking up requirements")
	if reqList.Err() != nil {
		return requirements, fmt.Errorf("failed to lookup requirements:\n%s", errors.Details(reqList.Err(), nil))
	}
	iter, err := reqList.List()
	if err != nil {
		return requirements, fmt.Errorf("failed to iterate requirements:\n%s", errors.Details(err, nil))
	}
	for iter.Next() {
		req := iter.Value()
		reqStr, err := req.String()
		if err != nil {
			return requirements, fmt.Errorf("failed to convert requirement to string:\n%s", errors.Details(err, nil))
		}
		requirements = append(requirements, reqStr)
	}
	return requirements, nil
}

// func checkToolExists validates the tool is installed on the system.
func checkToolExists(toolName string) error {
	_, err := exec.LookPath(toolName)
	return err
}

// func registerGaugeCmd registers the gauge command with the root command.
func registerGaugeCmd(rootCmd *cobra.Command) {
	gaugeCmd := &cobra.Command{
		Use:   "gauge",
		Short: "Gauge whether required tools are available.",
		RunE:  runGauge,
	}

	rootCmd.AddCommand(gaugeCmd)
}
