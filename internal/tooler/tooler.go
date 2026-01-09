package tooler

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

type Tooler interface {
	// Name of the tool.
	Name() string

	// Check whether the tool is available on the system.
	Gauge(context.Context) error

	// Installs the tool on the system.
	Install(context.Context) error
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
