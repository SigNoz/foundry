package convention

import (
	"strings"

	"github.com/signoz/foundry/internal/errors"
)

// Zone is an availability zone. String is the provider's identifier, kept
// verbatim; Short is the form a name carries, derived from it.
type Zone struct {
	s     string
	short string
}

// ParseZone drops the leading locale segment and joins the rest for the short
// form: "us-east-1a" becomes "east1a", "asia-south2-c" becomes "south2c".
func ParseZone(zone string) (Zone, error) {
	if zone == "" {
		return Zone{}, errors.Newf(errors.TypeInvalidInput, "failed to create zone from %q: zone is empty", zone)
	}

	segments := strings.Split(zone, "-")
	if len(segments) < 2 {
		return Zone{}, errors.Newf(errors.TypeInvalidInput, "failed to create zone from %q: zone has no locale and suffix segments", zone)
	}

	return Zone{s: zone, short: strings.Join(segments[1:], "")}, nil
}

func MustParseZone(zone string) Zone {
	parsed, err := ParseZone(zone)
	if err != nil {
		panic(err)
	}

	return parsed
}

func (zone Zone) String() string {
	return zone.s
}

func (zone Zone) Short() string {
	return zone.short
}
