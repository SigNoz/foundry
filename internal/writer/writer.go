package writer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/types"
)

type Options struct {
	Output io.Writer

	TargetDirectory string
}

type Writer struct {
	logger  *slog.Logger
	options Options
}

// NewManager creates a new output manager.
func New(logger *slog.Logger, options *Options) (*Writer, error) {
	if options == nil {
		options = &Options{
			Output:          &os.File{},
			TargetDirectory: "./pours",
		}
	}

	if options.Output == nil {
		options.Output = &os.File{}
	}

	if err := os.MkdirAll(options.TargetDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory '%s': %s", options.TargetDirectory, err.Error())
	}

	return &Writer{
		logger:  logger,
		options: *options,
	}, nil
}

func (w *Writer) Write(ctx context.Context, material types.Material) error {
	if _, ok := w.options.Output.(*os.File); ok {
		path := filepath.Join(w.options.TargetDirectory, material.Path())

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			w.logger.ErrorContext(ctx, "failed to create directory", slog.String("path", filepath.Dir(path)), foundryerrors.LogAttr(err))
			return err
		}

		if err := os.WriteFile(path, material.FmtContents(), 0644); err != nil {
			w.logger.ErrorContext(ctx, "failed to write material", slog.String("path", path), foundryerrors.LogAttr(err))
			return err
		}

		w.logger.InfoContext(ctx, "successfully wrote material", slog.String("path", path))
		return nil
	}

	_, err := w.options.Output.Write(material.FmtContents())
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to write material", foundryerrors.LogAttr(err))
		return err
	}

	w.logger.InfoContext(ctx, "successfully wrote material")
	return nil
}

func (w *Writer) WriteMany(ctx context.Context, materials ...types.Material) error {
	for _, material := range materials {
		if err := w.Write(ctx, material); err != nil {
			return err
		}
	}

	return nil
}
