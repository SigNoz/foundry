// Package updater surfaces a stderr notice when a newer foundryctl release is
// available on GitHub, following /releases/latest's redirect to read the tag.
package updater

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"golang.org/x/mod/semver"
)

const (
	baseURL     = "https://github.com"
	httpTimeout = 2 * time.Second

	// finishGrace caps the wait so short commands aren't perceptibly
	// slowed down when the network is sluggish.
	finishGrace = 300 * time.Millisecond

	noticeFormat = "\nA new release of foundryctl is available: %s → %s\nTo upgrade, run: curl -fsSL https://signoz.io/foundry.sh | bash\n%s\n\n"
)

// Release describes a foundryctl release on GitHub.
type Release struct {
	Version string
	URL     string
}

// Notifier runs an update check in the background. Empty releaseURL means
// inert — Notify and Finish are no-ops.
type Notifier struct {
	releaseURL string
	client     *http.Client
	logger     *slog.Logger

	ch     chan *Release
	cancel context.CancelFunc
}

// NewNotifier returns a Notifier wired to the foundry GitHub releases endpoint.
// releaseURL is built from the repository ldflag; absent or disabled, the
// Notifier is inert.
func NewNotifier(cfg Config, logger *slog.Logger) *Notifier {
	n := &Notifier{
		client: &http.Client{Timeout: httpTimeout},
		logger: logger,
	}
	if cfg.Enabled && repository != "" && repository != "<unset>" {
		n.releaseURL = baseURL + "/" + repository + "/releases/latest"
	}
	return n
}

// Notify launches the update check in a goroutine.
func (n *Notifier) Notify(ctx context.Context) {
	if n.releaseURL == "" {
		return
	}

	ctx, n.cancel = context.WithCancel(ctx)
	n.ch = make(chan *Release, 1)
	go func() {
		rel, err := n.fetchLatest(ctx)
		if err != nil {
			n.logger.DebugContext(ctx, "update check failed", foundryerrors.LogAttr(err))
		}
		n.ch <- rel
	}()
}

// Finish drains the in-flight check and writes a notice to w when the latest
// release is strictly newer than current. Cancels the goroutine's context on
// return so an unfinished HTTP request is torn down rather than leaking.
func (n *Notifier) Finish(current string, w io.Writer) {
	if n.ch == nil {
		return
	}
	defer n.cancel()

	var rel *Release
	select {
	case rel = <-n.ch:
	case <-time.After(finishGrace):
		return
	}

	if rel == nil || !isNewer(rel.Version, current) {
		return
	}

	_, _ = fmt.Fprintf(w, noticeFormat, current, rel.Version, rel.URL)
}

// fetchLatest follows GitHub's /releases/latest redirect to .../tag/<vX.Y.Z>
// and returns the resolved release.
func (n *Notifier) fetchLatest(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.releaseURL, nil)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "build latest-release request")
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "fetch latest release")
	}
	defer func() { _ = resp.Body.Close() }()

	finalURL := resp.Request.URL.String()
	tag, err := parseTagFromReleaseURL(finalURL)
	if err != nil {
		return nil, err
	}

	return &Release{Version: tag, URL: finalURL}, nil
}

// tagPattern mirrors the vMAJOR.MINOR.PATCH validation scripts/foundry.sh
// applies at line 179.
var tagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+`)

// parseTagFromReleaseURL extracts the release tag from a GitHub release URL
// like https://github.com/SigNoz/foundry/releases/tag/v0.1.4.
func parseTagFromReleaseURL(u string) (string, error) {
	const marker = "/releases/tag/"
	idx := strings.LastIndex(u, marker)
	if idx == -1 {
		return "", foundryerrors.Newf(foundryerrors.TypeInternal, "release URL has no /releases/tag/ segment: %s", u)
	}

	tag := u[idx+len(marker):]
	if !tagPattern.MatchString(tag) {
		return "", foundryerrors.Newf(foundryerrors.TypeInternal, "release tag is not vMAJOR.MINOR.PATCH: %s", tag)
	}

	return tag, nil
}

// isNewer reports whether latest is strictly newer than current. Returns
// false on either invalid input — when we can't tell, we don't warn.
func isNewer(latest, current string) bool {
	if !semver.IsValid(latest) || !semver.IsValid(current) {
		return false
	}
	return semver.Compare(latest, current) > 0
}
