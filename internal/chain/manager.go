package chain

import (
	"context"
	"fmt"
	"sync"

	"github.com/devdudeio/verus-gateway/internal/config"
	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/devdudeio/verus-gateway/pkg/verusrpc"
)

// Manager manages multiple blockchain connections
type Manager struct {
	chains       map[string]*Chain
	defaultChain string
	mu           sync.RWMutex
}

// Chain represents a configured blockchain with its RPC client
type Chain struct {
	ID     string
	Name   string
	Config config.ChainConfig
	Client *verusrpc.Client
}

// NewManager creates a new chain manager
func NewManager(cfg *config.Config) (*Manager, error) {
	manager := &Manager{
		chains:       make(map[string]*Chain),
		defaultChain: cfg.Chains.Default,
	}

	// Initialize all configured chains
	for id, chainCfg := range cfg.Chains.Chains {
		if !chainCfg.Enabled {
			continue
		}

		client := verusrpc.NewClient(verusrpc.Config{
			URL:         chainCfg.RPCURL,
			User:        chainCfg.RPCUser,
			Password:    chainCfg.RPCPassword,
			Timeout:     chainCfg.RPCTimeout,
			TLSInsecure: chainCfg.TLSInsecure,
			MaxRetries:  chainCfg.MaxRetries,
			RetryDelay:  chainCfg.RetryDelay,
		})

		chain := &Chain{
			ID:     id,
			Name:   chainCfg.Name,
			Config: chainCfg,
			Client: client,
		}

		manager.chains[id] = chain
	}

	if len(manager.chains) == 0 {
		return nil, fmt.Errorf("no chains configured")
	}

	// Validate default chain exists
	if manager.defaultChain != "" {
		if _, exists := manager.chains[manager.defaultChain]; !exists {
			return nil, fmt.Errorf("default chain %s not found", manager.defaultChain)
		}
	} else {
		// Set first chain as default
		for id := range manager.chains {
			manager.defaultChain = id
			break
		}
	}

	return manager, nil
}

// GetChain returns the RPC client for a specific chain
func (m *Manager) GetChain(chainID string) (*verusrpc.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, exists := m.chains[chainID]
	if !exists {
		return nil, domain.NewChainError(chainID, "chain not found")
	}

	return chain.Client, nil
}

// GetDefaultChain returns the default chain RPC client
func (m *Manager) GetDefaultChain() (*verusrpc.Client, error) {
	return m.GetChain(m.defaultChain)
}

// GetChainInfo returns chain information
func (m *Manager) GetChainInfo(chainID string) (*Chain, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chain, exists := m.chains[chainID]
	if !exists {
		return nil, domain.NewChainError(chainID, "chain not found")
	}

	return chain, nil
}

// ListChains returns all configured chain IDs
func (m *Manager) ListChains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]string, 0, len(m.chains))
	for id := range m.chains {
		chains = append(chains, id)
	}

	return chains
}

// HealthCheck checks if a chain is healthy
func (m *Manager) HealthCheck(ctx context.Context, chainID string) error {
	client, err := m.GetChain(chainID)
	if err != nil {
		return err
	}

	// Try to get chain info
	_, err = client.GetInfo(ctx)
	if err != nil {
		return domain.NewChainError(chainID, fmt.Sprintf("health check failed: %v", err))
	}

	return nil
}

// HealthCheckAll checks health of all chains
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]error {
	m.mu.RLock()
	chains := make([]string, 0, len(m.chains))
	for id := range m.chains {
		chains = append(chains, id)
	}
	m.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, chainID := range chains {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			err := m.HealthCheck(ctx, id)

			mu.Lock()
			results[id] = err
			mu.Unlock()
		}(chainID)
	}

	wg.Wait()
	return results
}

// GetDefaultChainID returns the ID of the default chain
func (m *Manager) GetDefaultChainID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultChain
}

// Close closes all chain connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for _, chain := range m.chains {
		if err := chain.Client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
