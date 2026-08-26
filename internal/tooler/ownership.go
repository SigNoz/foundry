package tooler

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

// Verify skips an unreadable ownership list rather than blocking: the verb
// itself then fails loudly, in the tool's own words.
func Verify(ctx context.Context, tool Tool, release domain.Release, list func(context.Context) (domain.Ownership, error)) error {
	ownership, err := list(ctx)
	if err != nil {
		tool.Logger.WarnContext(ctx, "skipping the ownership check: could not read labels", slog.String("tool", tool.Name()), errors.LogAttr(err))

		return nil
	}

	if foreign, conflict := ownership.Foreign(release.Owner); conflict {
		return errors.Newf(errors.TypeInvalidInput, "failed to run %s: %q already belongs to [%s], not [%s]: remove it, or deploy under a different name", tool.Name(), release.Name, foreign, release.Owner)
	}

	if ownership.HasUnowned() {
		tool.Logger.WarnContext(ctx, "release has objects without ownership labels", slog.String("tool", tool.Name()), slog.String("release", release.Name))
	}

	return nil
}
