package chain

import (
	"testing"
	"time"

	"github.com/devdudeio/verus-gateway/internal/config"
)

func TestNewManager_Success(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Default: "chain1",
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					RPCTimeout:  30 * time.Second,
					Enabled:     true,
				},
				"chain2": {
					Name:        "Chain 2",
					RPCURL:      "http://localhost:27487",
					RPCUser:     "user",
					RPCPassword: "pass",
					RPCTimeout:  30 * time.Second,
					Enabled:     true,
				},
			},
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if manager == nil {
		t.Fatal("manager is nil")
	}

	if len(manager.chains) != 2 {
		t.Errorf("expected 2 chains, got %d", len(manager.chains))
	}

	if manager.defaultChain != "chain1" {
		t.Errorf("default chain = %q, want chain1", manager.defaultChain)
	}
}

func TestNewManager_SkipsDisabledChains(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"enabled": {
					Name:        "Enabled Chain",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
				"disabled": {
					Name:        "Disabled Chain",
					RPCURL:      "http://localhost:27487",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     false,
				},
			},
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if len(manager.chains) != 1 {
		t.Errorf("expected 1 chain, got %d", len(manager.chains))
	}

	if _, exists := manager.chains["disabled"]; exists {
		t.Error("disabled chain should not be in manager")
	}
}

func TestNewManager_NoChains(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{},
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for no chains, got nil")
	}
}

func TestNewManager_InvalidDefaultChain(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Default: "nonexistent",
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for invalid default chain, got nil")
	}
}

func TestNewManager_AutoSelectDefault(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Default: "", // Empty default
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if manager.defaultChain == "" {
		t.Error("default chain should be auto-selected")
	}
}

func TestGetChain(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	client, err := manager.GetChain("chain1")
	if err != nil {
		t.Errorf("GetChain failed: %v", err)
	}

	if client == nil {
		t.Error("client is nil")
	}
}

func TestGetChain_NotFound(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	_, err := manager.GetChain("nonexistent")
	if err == nil {
		t.Error("expected error for non existent chain, got nil")
	}
}

func TestGetDefaultChain(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Default: "chain1",
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	client, err := manager.GetDefaultChain()
	if err != nil {
		t.Errorf("GetDefaultChain failed: %v", err)
	}

	if client == nil {
		t.Error("default client is nil")
	}
}

func TestGetChainInfo(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Test Chain",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	chain, err := manager.GetChainInfo("chain1")
	if err != nil {
		t.Errorf("GetChainInfo failed: %v", err)
	}

	if chain == nil {
		t.Fatal("chain info is nil")
	}

	if chain.Name != "Test Chain" {
		t.Errorf("chain name = %q, want Test Chain", chain.Name)
	}

	if chain.ID != "chain1" {
		t.Errorf("chain ID = %q, want chain1", chain.ID)
	}
}

func TestGetChainInfo_NotFound(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	_, err := manager.GetChainInfo("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent chain, got nil")
	}
}

func TestListChains(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
				"chain2": {
					Name:        "Chain 2",
					RPCURL:      "http://localhost:27487",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	chains := manager.ListChains()
	if len(chains) != 2 {
		t.Errorf("expected 2 chains, got %d", len(chains))
	}

	// Check both chains are in the list
	found := make(map[string]bool)
	for _, id := range chains {
		found[id] = true
	}

	if !found["chain1"] || !found["chain2"] {
		t.Error("not all chains returned from ListChains")
	}
}

func TestGetDefaultChainID(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Default: "mychain",
			Chains: map[string]config.ChainConfig{
				"mychain": {
					Name:        "My Chain",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	defaultID := manager.GetDefaultChainID()
	if defaultID != "mychain" {
		t.Errorf("default chain ID = %q, want mychain", defaultID)
	}
}

func TestClose(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	// Close should not error even if clients fail to close
	err := manager.Close()
	// We don't check for specific error since RPC clients may or may not error
	_ = err
}

func TestManager_ConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		Chains: config.ChainsConfig{
			Default: "chain1",
			Chains: map[string]config.ChainConfig{
				"chain1": {
					Name:        "Chain 1",
					RPCURL:      "http://localhost:27486",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
				"chain2": {
					Name:        "Chain 2",
					RPCURL:      "http://localhost:27487",
					RPCUser:     "user",
					RPCPassword: "pass",
					Enabled:     true,
				},
			},
		},
	}

	manager, _ := NewManager(cfg)

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			manager.GetChain("chain1")
			manager.GetChain("chain2")
			manager.ListChains()
			manager.GetDefaultChainID()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
