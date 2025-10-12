package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Create empty temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Write minimal valid config
	content := `
chains:
  chains:
    vrsctest:
      name: "Test Chain"
      enabled: true
      rpc_url: "http://localhost:18843"
      rpc_user: "test"
      rpc_password: "test"
      rpc_timeout: 10s
      max_retries: 3
      retry_delay: 100ms
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Default port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Default host = %s, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Cache.Type != "filesystem" {
		t.Errorf("Default cache type = %s, want filesystem", cfg.Cache.Type)
	}
	if cfg.Observability.Logging.Level != "info" {
		t.Errorf("Default log level = %s, want info", cfg.Observability.Logging.Level)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	content := `
server:
  port: 9090
  host: "127.0.0.1"
  read_timeout: 30s
  write_timeout: 120s

chains:
  default: vrsc
  chains:
    vrsc:
      name: "Verus Mainnet"
      enabled: true
      rpc_url: "http://localhost:27486"
      rpc_user: "user"
      rpc_password: "pass"
      rpc_timeout: 30s
      max_retries: 5
      retry_delay: 500ms

cache:
  type: redis
  ttl: 48h

observability:
  logging:
    level: debug
    format: json
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check custom values
	if cfg.Server.Port != 9090 {
		t.Errorf("Server port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server host = %s, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Cache.Type != "redis" {
		t.Errorf("Cache type = %s, want redis", cfg.Cache.Type)
	}
	if cfg.Cache.TTL != 48*time.Hour {
		t.Errorf("Cache TTL = %v, want 48h", cfg.Cache.TTL)
	}
	if cfg.Observability.Logging.Level != "debug" {
		t.Errorf("Log level = %s, want debug", cfg.Observability.Logging.Level)
	}
	if cfg.Chains.Default != "vrsc" {
		t.Errorf("Default chain = %s, want vrsc", cfg.Chains.Default)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 999999,
		},
		Chains: ChainsConfig{
			Chains: map[string]ChainConfig{
				"test": {
					Name:        "Test",
					Enabled:     true,
					RPCURL:      "http://localhost:8080",
					RPCUser:     "user",
					RPCPassword: "pass",
					RPCTimeout:  10 * time.Second,
					MaxRetries:  3,
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for invalid port, got nil")
	}
}

func TestValidate_NoChains(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Chains: ChainsConfig{
			Chains: map[string]ChainConfig{},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for no chains, got nil")
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Chains: ChainsConfig{
			Chains: map[string]ChainConfig{
				"test": {
					Name:        "Test",
					Enabled:     true,
					RPCURL:      "http://localhost:8080",
					RPCUser:     "user",
					RPCPassword: "pass",
					RPCTimeout:  10 * time.Second,
					MaxRetries:  3,
				},
			},
		},
		Observability: ObservabilityConfig{
			Logging: LoggingConfig{
				Level: "invalid",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for invalid log level, got nil")
	}
}

func TestChainConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ChainConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: ChainConfig{
				Name:        "Test",
				Enabled:     true,
				RPCURL:      "http://localhost:8080",
				RPCUser:     "user",
				RPCPassword: "pass",
				RPCTimeout:  10 * time.Second,
				MaxRetries:  3,
			},
			wantErr: false,
		},
		{
			name: "disabled chain skips validation",
			cfg: ChainConfig{
				Name:    "Test",
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing rpc url",
			cfg: ChainConfig{
				Name:        "Test",
				Enabled:     true,
				RPCUser:     "user",
				RPCPassword: "pass",
				RPCTimeout:  10 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "missing rpc user",
			cfg: ChainConfig{
				Name:        "Test",
				Enabled:     true,
				RPCURL:      "http://localhost:8080",
				RPCPassword: "pass",
				RPCTimeout:  10 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "timeout too short",
			cfg: ChainConfig{
				Name:        "Test",
				Enabled:     true,
				RPCURL:      "http://localhost:8080",
				RPCUser:     "user",
				RPCPassword: "pass",
				RPCTimeout:  500 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "too many retries",
			cfg: ChainConfig{
				Name:        "Test",
				Enabled:     true,
				RPCURL:      "http://localhost:8080",
				RPCUser:     "user",
				RPCPassword: "pass",
				RPCTimeout:  10 * time.Second,
				MaxRetries:  15,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("ChainConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	// Try to load non-existent file - should error
	_, err := Load("nonexistent.yaml")

	// Should error because validation fails (no chains configured)
	if err == nil {
		t.Error("Load() expected error for non-existent file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Write invalid YAML
	content := `
invalid yaml content
  this is not: valid: yaml:
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}
