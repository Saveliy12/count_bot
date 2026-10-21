package env

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	timeZoneEnvName = "TIMEZONE"
	defaultTimeZone = "Europe/Moscow"
)

func NewTimeZone() (*time.Location, error) {
	name := strings.TrimSpace(os.Getenv(timeZoneEnvName))
	if name == "" {
		name = defaultTimeZone
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid TIMEZONE %q: %w", name, err)
	}

	return location, nil
}
