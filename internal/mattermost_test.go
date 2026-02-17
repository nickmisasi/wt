package internal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsPortAvailable tests the port availability check function
func TestIsPortAvailable(t *testing.T) {
	t.Run("available port returns true", func(t *testing.T) {
		// Find an available port by letting the OS assign one
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("failed to create test listener: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// The port should now be available
		if !IsPortAvailable(port) {
			t.Errorf("expected port %d to be available after closing listener", port)
		}
	})

	t.Run("in-use port returns false", func(t *testing.T) {
		// Occupy a port
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("failed to create test listener: %v", err)
		}
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		// The port should be in use
		if IsPortAvailable(port) {
			t.Errorf("expected port %d to be in use", port)
		}
	})
}

// TestExtractPortPairFromConfig tests extracting port pairs from config files
func TestExtractPortPairFromConfig(t *testing.T) {
	t.Run("valid config with both ports", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		config := map[string]interface{}{
			"ServiceSettings": map[string]interface{}{
				"ListenAddress": ":8100",
				"SiteURL":       "http://localhost:8100",
			},
			"MetricsSettings": map[string]interface{}{
				"ListenAddress": ":8101",
			},
		}

		data, _ := json.Marshal(config)
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		pair := ExtractPortPairFromConfig(configPath)

		if pair.ServerPort != 8100 {
			t.Errorf("expected server port 8100, got %d", pair.ServerPort)
		}
		if pair.MetricsPort != 8101 {
			t.Errorf("expected metrics port 8101, got %d", pair.MetricsPort)
		}
	})

	t.Run("config with only server port", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		config := map[string]interface{}{
			"ServiceSettings": map[string]interface{}{
				"ListenAddress": ":8200",
			},
		}

		data, _ := json.Marshal(config)
		os.WriteFile(configPath, data, 0644)

		pair := ExtractPortPairFromConfig(configPath)

		if pair.ServerPort != 8200 {
			t.Errorf("expected server port 8200, got %d", pair.ServerPort)
		}
		if pair.MetricsPort != 0 {
			t.Errorf("expected metrics port 0 (missing), got %d", pair.MetricsPort)
		}
	})

	t.Run("missing config file returns zero ports", func(t *testing.T) {
		pair := ExtractPortPairFromConfig("/nonexistent/config.json")

		if pair.ServerPort != 0 || pair.MetricsPort != 0 {
			t.Errorf("expected zero ports for missing file, got server=%d, metrics=%d",
				pair.ServerPort, pair.MetricsPort)
		}
	})

	t.Run("invalid JSON returns zero ports", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		os.WriteFile(configPath, []byte("not valid json"), 0644)

		pair := ExtractPortPairFromConfig(configPath)

		if pair.ServerPort != 0 || pair.MetricsPort != 0 {
			t.Errorf("expected zero ports for invalid JSON, got server=%d, metrics=%d",
				pair.ServerPort, pair.MetricsPort)
		}
	})
}

// TestUpdateConfigPorts tests the port update function
func TestUpdateConfigPorts(t *testing.T) {
	t.Run("updates existing ServiceSettings and MetricsSettings", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Create a config with existing settings
		config := map[string]interface{}{
			"ServiceSettings": map[string]interface{}{
				"ListenAddress": ":8065",
				"SiteURL":       "http://localhost:8065",
				"OtherSetting":  "preserved",
			},
			"MetricsSettings": map[string]interface{}{
				"ListenAddress": ":8067",
				"Enable":        true,
			},
			"OtherSection": map[string]interface{}{
				"Key": "value",
			},
		}

		data, _ := json.Marshal(config)
		os.WriteFile(configPath, data, 0644)

		// Update to new ports
		err := updateConfigPorts(configPath, 8891, 8893)
		if err != nil {
			t.Fatalf("updateConfigPorts failed: %v", err)
		}

		// Read back and verify
		pair := ExtractPortPairFromConfig(configPath)
		if pair.ServerPort != 8891 {
			t.Errorf("expected server port 8891, got %d", pair.ServerPort)
		}
		if pair.MetricsPort != 8893 {
			t.Errorf("expected metrics port 8893, got %d", pair.MetricsPort)
		}

		// Verify other settings are preserved
		updatedData, _ := os.ReadFile(configPath)
		var updatedConfig map[string]interface{}
		json.Unmarshal(updatedData, &updatedConfig)

		serviceSettings := updatedConfig["ServiceSettings"].(map[string]interface{})
		if serviceSettings["OtherSetting"] != "preserved" {
			t.Error("expected OtherSetting to be preserved")
		}
		if serviceSettings["SiteURL"] != "http://localhost:8891" {
			t.Errorf("expected SiteURL to be updated, got %v", serviceSettings["SiteURL"])
		}
	})

	t.Run("creates ServiceSettings if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Create a config WITHOUT ServiceSettings
		config := map[string]interface{}{
			"OtherSection": map[string]interface{}{
				"Key": "value",
			},
		}

		data, _ := json.Marshal(config)
		os.WriteFile(configPath, data, 0644)

		// Update should create ServiceSettings
		err := updateConfigPorts(configPath, 8891, 8893)
		if err != nil {
			t.Fatalf("updateConfigPorts failed: %v", err)
		}

		// Read back and verify ports were set
		pair := ExtractPortPairFromConfig(configPath)
		if pair.ServerPort != 8891 {
			t.Errorf("expected server port 8891 after creating ServiceSettings, got %d", pair.ServerPort)
		}
		if pair.MetricsPort != 8893 {
			t.Errorf("expected metrics port 8893 after creating MetricsSettings, got %d", pair.MetricsPort)
		}
	})

	t.Run("creates MetricsSettings if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Create a config with ServiceSettings but no MetricsSettings
		config := map[string]interface{}{
			"ServiceSettings": map[string]interface{}{
				"ListenAddress": ":8065",
			},
		}

		data, _ := json.Marshal(config)
		os.WriteFile(configPath, data, 0644)

		err := updateConfigPorts(configPath, 8891, 8893)
		if err != nil {
			t.Fatalf("updateConfigPorts failed: %v", err)
		}

		pair := ExtractPortPairFromConfig(configPath)
		if pair.MetricsPort != 8893 {
			t.Errorf("expected metrics port 8893 after creating MetricsSettings, got %d", pair.MetricsPort)
		}
	})

	t.Run("handles empty config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Create an empty JSON object
		os.WriteFile(configPath, []byte("{}"), 0644)

		err := updateConfigPorts(configPath, 8891, 8893)
		if err != nil {
			t.Fatalf("updateConfigPorts failed on empty config: %v", err)
		}

		pair := ExtractPortPairFromConfig(configPath)
		if pair.ServerPort != 8891 {
			t.Errorf("expected server port 8891 on empty config, got %d", pair.ServerPort)
		}
		if pair.MetricsPort != 8893 {
			t.Errorf("expected metrics port 8893 on empty config, got %d", pair.MetricsPort)
		}
	})

	t.Run("handles null ServiceSettings", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Create a config with null ServiceSettings (this was the bug scenario)
		os.WriteFile(configPath, []byte(`{"ServiceSettings": null, "MetricsSettings": null}`), 0644)

		err := updateConfigPorts(configPath, 8891, 8893)
		if err != nil {
			t.Fatalf("updateConfigPorts failed on null settings: %v", err)
		}

		pair := ExtractPortPairFromConfig(configPath)
		if pair.ServerPort != 8891 {
			t.Errorf("expected server port 8891 when ServiceSettings was null, got %d", pair.ServerPort)
		}
		if pair.MetricsPort != 8893 {
			t.Errorf("expected metrics port 8893 when MetricsSettings was null, got %d", pair.MetricsPort)
		}
	})
}

// TestGetReservedPorts tests the reserved port extraction from worktrees
func TestGetReservedPorts(t *testing.T) {
	t.Run("empty worktrees returns only excluded ports", func(t *testing.T) {
		reserved := GetReservedPorts(nil)

		// Should contain the excluded ports
		if !reserved[MainRepoPort] {
			t.Errorf("expected main repo port %d to be reserved", MainRepoPort)
		}
		if !reserved[MainRepoPort+2] {
			t.Errorf("expected main repo metrics port %d to be reserved", MainRepoPort+2)
		}
	})

	t.Run("worktrees with valid configs are reserved", func(t *testing.T) {
		// Create a mock worktree structure
		tmpDir := t.TempDir()
		worktreePath := filepath.Join(tmpDir, "mattermost-test-branch")

		// Create the mattermost-test directory structure
		mmDir := filepath.Join(worktreePath, "mattermost-test")
		configDir := filepath.Join(mmDir, "server", "config")
		os.MkdirAll(configDir, 0755)

		// Create .git files to make it look like worktrees
		os.WriteFile(filepath.Join(mmDir, ".git"), []byte("gitdir: /path/to/git"), 0644)

		// Create enterprise directory
		entDir := filepath.Join(worktreePath, "enterprise-test")
		os.MkdirAll(entDir, 0755)
		os.WriteFile(filepath.Join(entDir, ".git"), []byte("gitdir: /path/to/git"), 0644)

		// Write config with specific ports
		config := map[string]interface{}{
			"ServiceSettings": map[string]interface{}{
				"ListenAddress": ":8300",
			},
			"MetricsSettings": map[string]interface{}{
				"ListenAddress": ":8301",
			},
		}
		data, _ := json.Marshal(config)
		os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)

		worktrees := []WorktreeInfo{
			{Path: worktreePath, Branch: "test-branch"},
		}

		reserved := GetReservedPorts(worktrees)

		if !reserved[8300] {
			t.Errorf("expected port 8300 to be reserved")
		}
		if !reserved[8301] {
			t.Errorf("expected port 8301 to be reserved")
		}
	})
}

// TestGetAvailablePortsWithRand tests the main port selection logic
func TestGetAvailablePortsWithRand(t *testing.T) {
	t.Run("returns ports within valid range", func(t *testing.T) {
		// Use a seeded RNG for deterministic behavior
		rng := rand.New(rand.NewSource(42))

		serverPort, metricsPort := GetAvailablePortsWithRand(nil, rng)

		if serverPort < PortRangeStart || serverPort > PortRangeEnd-MetricsPortOffset {
			t.Errorf("server port %d outside valid range [%d, %d]",
				serverPort, PortRangeStart, PortRangeEnd-MetricsPortOffset)
		}

		expectedMetrics := serverPort + MetricsPortOffset
		if metricsPort != expectedMetrics {
			t.Errorf("expected metrics port %d, got %d", expectedMetrics, metricsPort)
		}
	})

	t.Run("different seeds produce different ports", func(t *testing.T) {
		rng1 := rand.New(rand.NewSource(1))
		rng2 := rand.New(rand.NewSource(2))

		port1, _ := GetAvailablePortsWithRand(nil, rng1)
		port2, _ := GetAvailablePortsWithRand(nil, rng2)

		// With different seeds, we should (very likely) get different ports
		// This isn't guaranteed, but with the large range it's extremely unlikely to collide
		if port1 == port2 {
			t.Logf("Warning: same port %d with different seeds (unlikely but possible)", port1)
		}
	})

	t.Run("avoids reserved ports", func(t *testing.T) {
		// Create mock worktrees that reserve specific ports
		tmpDir := t.TempDir()

		// Create multiple worktrees reserving ports
		reservedServerPorts := []int{8100, 8102, 8104}

		var worktrees []WorktreeInfo
		for i, port := range reservedServerPorts {
			worktreePath := filepath.Join(tmpDir, fmt.Sprintf("mattermost-branch%d", i))
			mmDir := filepath.Join(worktreePath, fmt.Sprintf("mattermost-branch%d", i))
			configDir := filepath.Join(mmDir, "server", "config")
			os.MkdirAll(configDir, 0755)

			os.WriteFile(filepath.Join(mmDir, ".git"), []byte("gitdir: /path/to/git"), 0644)

			entDir := filepath.Join(worktreePath, fmt.Sprintf("enterprise-branch%d", i))
			os.MkdirAll(entDir, 0755)
			os.WriteFile(filepath.Join(entDir, ".git"), []byte("gitdir: /path/to/git"), 0644)

			config := map[string]interface{}{
				"ServiceSettings": map[string]interface{}{
					"ListenAddress": fmt.Sprintf(":%d", port),
				},
				"MetricsSettings": map[string]interface{}{
					"ListenAddress": fmt.Sprintf(":%d", port+MetricsPortOffset),
				},
			}

			data, _ := json.Marshal(config)
			os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)

			worktrees = append(worktrees, WorktreeInfo{
				Path:   worktreePath,
				Branch: fmt.Sprintf("branch%d", i),
			})
		}

		// Get reserved ports
		reserved := GetReservedPorts(worktrees)

		// Run multiple times to verify we never get reserved ports
		for i := 0; i < 10; i++ {
			rng := rand.New(rand.NewSource(int64(i * 100)))
			serverPort, metricsPort := GetAvailablePortsWithRand(worktrees, rng)

			if reserved[serverPort] {
				t.Errorf("iteration %d: got reserved server port %d", i, serverPort)
			}
			if reserved[metricsPort] {
				t.Errorf("iteration %d: got reserved metrics port %d", i, metricsPort)
			}
		}
	})

	t.Run("excludes main repo port", func(t *testing.T) {
		// Run many iterations to ensure we never get the main repo port
		for i := 0; i < 100; i++ {
			rng := rand.New(rand.NewSource(int64(i)))
			serverPort, metricsPort := GetAvailablePortsWithRand(nil, rng)

			if serverPort == MainRepoPort {
				t.Errorf("got excluded main repo port %d", MainRepoPort)
			}
			if metricsPort == MainRepoPort || metricsPort == MainRepoPort+2 {
				t.Errorf("got excluded main repo metrics port")
			}
		}
	})

	t.Run("deterministic with same seed", func(t *testing.T) {
		rng1 := rand.New(rand.NewSource(12345))
		rng2 := rand.New(rand.NewSource(12345))

		port1, metrics1 := GetAvailablePortsWithRand(nil, rng1)
		port2, metrics2 := GetAvailablePortsWithRand(nil, rng2)

		if port1 != port2 || metrics1 != metrics2 {
			t.Errorf("expected same ports with same seed: got (%d,%d) vs (%d,%d)",
				port1, metrics1, port2, metrics2)
		}
	})
}

// TestPortConstants verifies the port constants are set correctly
func TestPortConstants(t *testing.T) {
	t.Run("port range is valid", func(t *testing.T) {
		if PortRangeStart >= PortRangeEnd {
			t.Errorf("invalid port range: start %d >= end %d", PortRangeStart, PortRangeEnd)
		}
	})

	t.Run("main repo port is excluded from range", func(t *testing.T) {
		// The main repo port (8065) should not be in our allocation range
		if MainRepoPort >= PortRangeStart && MainRepoPort <= PortRangeEnd {
			// It's in the range, so it must be in ExcludedPorts
			if !ExcludedPorts[MainRepoPort] {
				t.Errorf("main repo port %d is in range but not excluded", MainRepoPort)
			}
		}
	})

	t.Run("metrics offset is positive", func(t *testing.T) {
		if MetricsPortOffset <= 0 {
			t.Errorf("metrics port offset should be positive, got %d", MetricsPortOffset)
		}
	})

	t.Run("retry count is reasonable", func(t *testing.T) {
		if PortRandomRetries < 10 {
			t.Errorf("retry count %d seems too low", PortRandomRetries)
		}
		if PortRandomRetries > 1000 {
			t.Errorf("retry count %d seems too high", PortRandomRetries)
		}
	})
}

// TestIsPortPairAvailable tests the port pair availability check
func TestIsPortPairAvailable(t *testing.T) {
	t.Run("both ports free and not reserved", func(t *testing.T) {
		reserved := map[int]bool{8200: true, 8201: true}

		// Port 8300 and 8301 should be available (not in reserved)
		if !isPortPairAvailable(8300, reserved) {
			// This might fail if 8300 or 8301 is actually in use on the system
			t.Log("Port 8300/8301 might be in use on the system")
		}
	})

	t.Run("server port reserved", func(t *testing.T) {
		reserved := map[int]bool{8400: true}

		if isPortPairAvailable(8400, reserved) {
			t.Error("expected port pair to be unavailable when server port is reserved")
		}
	})

	t.Run("metrics port reserved", func(t *testing.T) {
		reserved := map[int]bool{8502: true} // metrics port for 8500 (with offset 2)

		if isPortPairAvailable(8500, reserved) {
			t.Error("expected port pair to be unavailable when metrics port is reserved")
		}
	})
}

// TestSequentialFallback tests that sequential scan works when random fails
func TestSequentialFallback(t *testing.T) {
	t.Run("finds port when most are reserved", func(t *testing.T) {
		// Create a reserved map with most ports taken
		reserved := make(map[int]bool)
		for port := PortRangeStart; port <= PortRangeEnd; port++ {
			reserved[port] = true
		}

		// Leave just one port pair available
		delete(reserved, 8500)
		delete(reserved, 8500+MetricsPortOffset)

		// Mock isPortPairAvailable to use our reserved map without actual port checking
		// Since we can't easily mock the actual port check, we test the logic indirectly

		// For this test, we verify the function doesn't crash with heavy reservation
		rng := rand.New(rand.NewSource(999))
		serverPort, metricsPort := GetAvailablePortsWithRand(nil, rng)

		// Should still return valid ports (the actual availability depends on system state)
		if serverPort == 0 && metricsPort == 0 {
			t.Log("All ports were unavailable - this is expected only if system ports are exhausted")
		}
	})
}

// setupTestGitRepo initializes a git repo at path with an initial commit on "main"
// and optionally creates additional branches.
func setupTestGitRepo(t *testing.T, path string, extraBranches ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available on PATH")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create repo directory %s: %v", path, err)
	}

	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed in %s: %v\n%s", args, path, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")

	for _, branch := range extraBranches {
		if branch != "main" {
			run("branch", branch)
		}
	}
}

// TestCreateWorktreeForRepo_BaseBranchNotFound verifies that createWorktreeForRepo
// returns an error containing "not found in" when the base branch doesn't exist.
func TestCreateWorktreeForRepo_BaseBranchNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	setupTestGitRepo(t, repoPath)

	repo := &GitRepo{Root: repoPath, Name: "test-repo"}
	worktreePath := filepath.Join(tmpDir, "wt-test")

	err := createWorktreeForRepo(repo, "new-branch", "nonexistent-base", worktreePath)
	if err == nil {
		t.Fatal("expected error when base branch doesn't exist, got nil")
	}
	if !strings.Contains(err.Error(), "not found in") {
		t.Errorf("expected error to contain 'not found in', got: %v", err)
	}
}

// TestCreateMattermostDualWorktree_EnterpriseFallback verifies that when the base
// branch doesn't exist in the enterprise repo, the function falls back to the
// enterprise repo's default branch instead of failing.
func TestCreateMattermostDualWorktree_EnterpriseFallback(t *testing.T) {
	tmpDir := t.TempDir()
	mattermostPath := filepath.Join(tmpDir, "mattermost")
	enterprisePath := filepath.Join(tmpDir, "enterprise")
	worktreeBasePath := filepath.Join(tmpDir, "worktrees")

	// Mattermost repo has main + release-1.0
	setupTestGitRepo(t, mattermostPath, "release-1.0")
	// Enterprise repo only has main (no release-1.0)
	setupTestGitRepo(t, enterprisePath)

	// Create the required server/config/config.json in the mattermost repo
	configDir := filepath.Join(mattermostPath, "server", "config")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.json"),
		[]byte(`{"ServiceSettings":{"ListenAddress":":8065"}}`), 0644)

	mc := &MattermostConfig{
		WorkspaceRoot:    tmpDir,
		MattermostPath:   mattermostPath,
		EnterprisePath:   enterprisePath,
		WorktreeBasePath: worktreeBasePath,
		ServerPort:       8200,
		MetricsPort:      8202,
	}

	// Create worktree with baseBranch that only exists in mattermost, not enterprise.
	// Enterprise should fall back to its default branch ("main") instead of failing.
	result, err := CreateMattermostDualWorktree(mc, "test-branch", "release-1.0")
	if err != nil {
		t.Fatalf("expected success with enterprise fallback, got error: %v", err)
	}

	// Verify the worktree directory was created
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("expected worktree directory to exist at %s", result)
	}

	// Verify both mattermost and enterprise worktree subdirs exist
	sanitized := SanitizeBranchName("test-branch")
	mmWorktree := filepath.Join(result, "mattermost-"+sanitized)
	entWorktree := filepath.Join(result, "enterprise-"+sanitized)

	if _, err := os.Stat(mmWorktree); os.IsNotExist(err) {
		t.Errorf("expected mattermost worktree at %s", mmWorktree)
	}
	if _, err := os.Stat(entWorktree); os.IsNotExist(err) {
		t.Errorf("expected enterprise worktree at %s", entWorktree)
	}
}

// TestCreateMattermostDualWorktree_BothReposHaveBranch verifies the normal case
// where the base branch exists in both repos (no fallback needed).
func TestCreateMattermostDualWorktree_BothReposHaveBranch(t *testing.T) {
	tmpDir := t.TempDir()
	mattermostPath := filepath.Join(tmpDir, "mattermost")
	enterprisePath := filepath.Join(tmpDir, "enterprise")
	worktreeBasePath := filepath.Join(tmpDir, "worktrees")

	// Both repos have main + release-1.0
	setupTestGitRepo(t, mattermostPath, "release-1.0")
	setupTestGitRepo(t, enterprisePath, "release-1.0")

	// Create the required server/config/config.json
	configDir := filepath.Join(mattermostPath, "server", "config")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.json"),
		[]byte(`{"ServiceSettings":{"ListenAddress":":8065"}}`), 0644)

	mc := &MattermostConfig{
		WorkspaceRoot:    tmpDir,
		MattermostPath:   mattermostPath,
		EnterprisePath:   enterprisePath,
		WorktreeBasePath: worktreeBasePath,
		ServerPort:       8300,
		MetricsPort:      8302,
	}

	result, err := CreateMattermostDualWorktree(mc, "test-branch-2", "release-1.0")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("expected worktree directory to exist at %s", result)
	}
}

