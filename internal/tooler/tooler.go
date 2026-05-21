package tooler

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"

	foundryerrors "github.com/signoz/foundry/internal/errors"
)

type Tooler interface {
	// Name of the tool.
	Name() string

	// Check whether the tool is available on the system.
	Gauge(context.Context) error

	// Installs the tool on the system.
	Install(context.Context) error
}

// GaugeAll gauges every Tooler in toolers, logging per-tool availability.
// Returns a TypeNotFound error listing the tools that could not be detected;
// nil if every tool is available.
func GaugeAll(ctx context.Context, logger *slog.Logger, toolers []Tooler) error {
	var unavailable []string
	for _, t := range toolers {
		if err := t.Gauge(ctx); err != nil {
			logger.ErrorContext(ctx, "tool is not available or cannot be detected properly", slog.String("tool.name", t.Name()), foundryerrors.LogAttr(err))
			unavailable = append(unavailable, t.Name())
			continue
		}
		logger.InfoContext(ctx, "tool is available", slog.String("tool.name", t.Name()))
	}
	if len(unavailable) > 0 {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "tools are not available, please install them and try again: %s", strings.Join(unavailable, ", "))
	}
	return nil
}

func ExecChecker(ctx context.Context, toolName string) error {
	_, err := exec.LookPath(toolName)
	return err
}

func MultiExecChecker(ctx context.Context, toolNames ...string) error {
	var errs []error

	for _, toolName := range toolNames {
		if err := ExecChecker(ctx, toolName); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func AnyOneExecChecker(ctx context.Context, toolNames ...string) error {
	for _, toolName := range toolNames {
		if err := ExecChecker(ctx, toolName); err == nil {
			return nil
		}
	}

	return errors.New("none of the tools '" + strings.Join(toolNames, ", ") + "' are available on the system")
}
