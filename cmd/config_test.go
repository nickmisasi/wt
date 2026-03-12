package cmd

import (
	"strings"
	"testing"
)

func TestConfigUsageContainsClaudemuxKeys(t *testing.T) {
	keys := []string{
		"claudemux.enabled",
		"claudemux.command",
		"claudemux.max_sessions",
	}

	for _, key := range keys {
		if !strings.Contains(configUsage, key) {
			t.Errorf("configUsage missing key: %s", key)
		}
	}
}
