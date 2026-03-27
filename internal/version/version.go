package version

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// These will be set via ldflags at build time.
var (
	version string = "<unset>"
	hash    string = "<unset>"
)

var (
	Info Build = Build{
		version:   version,
		hash:      hash,
		goVersion: runtime.Version(),
	}
)

// Build contains information about the build environment.
type Build struct {
	// The version of the current build.
	version string

	// The git hash of the current build.
	hash string

	// The version of go.
	goVersion string
}

func (b Build) Version() string {
	return b.version
}

func (b Build) Hash() string {
	return b.hash
}

func (b Build) GoVersion() string {
	return b.goVersion
}

func (b Build) Short() string {
	return fmt.Sprintf("foundryctl %s (%s) %s", b.version, b.hash, b.goVersion)
}

func (b Build) PrettyPrint() {
	year := time.Now().Year()
	ascii := []string{
		"███████╗ ██████╗ ██╗   ██╗███╗   ██╗██████╗ ██████╗ ██╗   ██╗",
		"██╔════╝██╔═══██╗██║   ██║████╗  ██║██╔══██╗██╔══██╗╚██╗ ██╔╝",
		"█████╗  ██║   ██║██║   ██║██╔██╗ ██║██║  ██║██████╔╝ ╚████╔╝ ",
		"██╔══╝  ██║   ██║██║   ██║██║╚██╗██║██║  ██║██╔══██╗  ╚██╔╝  ",
		"██║     ╚██████╔╝╚██████╔╝██║ ╚████║██████╔╝██║  ██║   ██║   ",
		"╚═╝      ╚═════╝  ╚═════╝ ╚═╝  ╚═══╝╚═════╝ ╚═╝  ╚═╝   ╚═╝   ",
	}

	_, _ = fmt.Fprintln(os.Stdout)
	for _, line := range ascii {
		_, _ = fmt.Fprintln(os.Stdout, line)
	}
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintf(os.Stdout, "  Version:  %s\n", b.version)
	_, _ = fmt.Fprintf(os.Stdout, "  Commit:   %s\n", b.hash)
	_, _ = fmt.Fprintf(os.Stdout, "  Go:       %s\n", b.goVersion)
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintf(os.Stdout, "  Copyright %d SigNoz, All rights reserved.\n", year)
	_, _ = fmt.Fprintln(os.Stdout)
}
