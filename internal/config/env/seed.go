package env

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const seedDemoUserEnvName = "SEED_DEMO_USER"

func NewSeedDemoUser() (bool, error) {
	raw := strings.TrimSpace(os.Getenv(seedDemoUserEnvName))
	if raw == "" {
		return false, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, errors.New("SEED_DEMO_USER must be a boolean")
	}

	return value, nil
}
