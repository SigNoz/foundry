package update

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotifier(t *testing.T) {
	tests := []struct {
		name               string
		wireURL            bool
		upstream           string
		current            string
		expectedSubstrings []string
	}{
		{
			name:    "Inert_NoURL_Silent",
			wireURL: false,
			current: "v0.1.3",
		},
		{
			name:     "UpstreamNewer_Prints",
			wireURL:  true,
			upstream: "v0.2.0",
			current:  "v0.1.3",
			expectedSubstrings: []string{
				"v0.1.3 → v0.2.0",
				"curl -fsSL https://signoz.io/foundry.sh | bash",
				"/releases/tag/v0.2.0",
			},
		},
		{
			name:     "UpToDate_Silent",
			wireURL:  true,
			upstream: "v0.1.3",
			current:  "v0.1.3",
		},
		{
			name:     "AheadOfUpstream_Silent",
			wireURL:  true,
			upstream: "v0.1.0",
			current:  "v0.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/SigNoz/foundry/releases/latest", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/SigNoz/foundry/releases/tag/"+tt.upstream, http.StatusFound)
			})
			mux.HandleFunc("/SigNoz/foundry/releases/tag/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			ts := httptest.NewServer(mux)
			t.Cleanup(ts.Close)

			buf := &bytes.Buffer{}
			n := NewNotifier(NewConfig(), slog.New(slog.DiscardHandler))
			if tt.wireURL {
				n.releaseURL = ts.URL + "/SigNoz/foundry/releases/latest"
				n.client = ts.Client()
			}

			n.Notify(context.Background())
			n.Finish(tt.current, buf)

			out := buf.String()
			if len(tt.expectedSubstrings) == 0 {
				assert.Empty(t, out)
				return
			}

			for _, s := range tt.expectedSubstrings {
				assert.Contains(t, out, s)
			}
		})
	}
}

func TestParseTagFromReleaseURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		pass     bool
		expected string
	}{
		{
			name:     "Valid",
			url:      "https://github.com/SigNoz/foundry/releases/tag/v0.1.4",
			pass:     true,
			expected: "v0.1.4",
		},
		{
			name:     "Prerelease_Valid",
			url:      "https://github.com/SigNoz/foundry/releases/tag/v1.2.3-rc.1",
			pass:     true,
			expected: "v1.2.3-rc.1",
		},
		{
			name: "NoMarker_Invalid",
			url:  "https://github.com/SigNoz/foundry/releases",
			pass: false,
		},
		{
			name: "InvalidShape_Invalid",
			url:  "https://github.com/SigNoz/foundry/releases/tag/latest",
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := parseTagFromReleaseURL(tt.url)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tag)
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name     string
		latest   string
		current  string
		expected bool
	}{
		{name: "Newer", latest: "v0.2.0", current: "v0.1.3", expected: true},
		{name: "Equal", latest: "v0.1.3", current: "v0.1.3", expected: false},
		{name: "Older", latest: "v0.1.0", current: "v0.1.3", expected: false},
		{name: "Invalid", latest: "not-semver", current: "v0.1.3", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNewer(tt.latest, tt.current))
		})
	}
}
