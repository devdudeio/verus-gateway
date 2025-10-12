package verusrpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// Client is a Verus RPC client
type Client struct {
	url        string
	user       string
	password   string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	retryDelay time.Duration

	// Metrics
	requestCount  atomic.Uint64
	errorCount    atomic.Uint64
	totalDuration atomic.Int64 // in microseconds
}

// Config holds configuration for the RPC client
type Config struct {
	URL         string
	User        string
	Password    string
	Timeout     time.Duration
	TLSInsecure bool
	MaxRetries  int
	RetryDelay  time.Duration
}

// NewClient creates a new Verus RPC client
func NewClient(cfg Config) *Client {
	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 500 * time.Millisecond
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSInsecure, // #nosec G402
		},
	}

	return &Client{
		url:      cfg.URL,
		user:     cfg.User,
		password: cfg.Password,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
		timeout:    cfg.Timeout,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
	}
}

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Call makes a JSON-RPC call
func (c *Client) Call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		result, err := c.call(ctx, method, params...)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on context errors or certain RPC errors
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Don't retry on client errors (4xx)
		if rpcErr, ok := err.(*RPCError); ok {
			if rpcErr.Code >= -32099 && rpcErr.Code <= -32000 {
				// Standard JSON-RPC errors, don't retry
				return nil, err
			}
		}

		// Retry on network errors and server errors
	}

	return nil, fmt.Errorf("rpc call failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// call makes a single JSON-RPC call
func (c *Client) call(ctx context.Context, method string, params ...interface{}) (json.RawMessage, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.recordMetrics(duration, nil)
	}()

	// Increment request count
	c.requestCount.Add(1)

	// Create request
	reqBody := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.errorCount.Add(1)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewReader(jsonData))
	if err != nil {
		c.errorCount.Add(1)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.user, c.password)

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.errorCount.Add(1)
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.errorCount.Add(1)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		c.errorCount.Add(1)
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON-RPC response
	var rpcResp Response
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		c.errorCount.Add(1)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for RPC error
	if rpcResp.Error != nil {
		c.errorCount.Add(1)
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// DecryptData calls the decryptdata RPC method
func (c *Client) DecryptData(ctx context.Context, txid, evk string) (string, error) {
	// Build the request object with datadescriptor structure
	// This structure is required by Verus for file decryption
	params := map[string]interface{}{
		"datadescriptor": map[string]interface{}{
			"version": 1,
			"flags":   0,
			"objectdata": map[string]interface{}{
				"iP3euVSzNcXUrLNHnQnR9G6q8jeYuGSxgw": map[string]interface{}{
					"type":      0,
					"version":   1,
					"flags":     1,
					"output":    map[string]interface{}{"txid": "0000000000000000000000000000000000000000000000000000000000000000", "voutnum": 0},
					"objectnum": 0,
					"subobject": 0,
				},
			},
		},
		"txid":     txid,
		"retrieve": true,
		"evk":      evk,
	}

	result, err := c.Call(ctx, "decryptdata", params)
	if err != nil {
		return "", fmt.Errorf("decryptdata failed: %w", err)
	}

	// Parse the response structure:
	// result is an array of objects, each with objectdata field containing hex-encoded data
	var resultArray []map[string]interface{}
	if err := json.Unmarshal(result, &resultArray); err != nil {
		return "", fmt.Errorf("failed to parse decryptdata result array: %w", err)
	}

	if len(resultArray) == 0 {
		return "", fmt.Errorf("decryptdata returned empty result")
	}

	// Extract objectdata field from first element
	objectData, ok := resultArray[0]["objectdata"].(string)
	if !ok {
		return "", fmt.Errorf("objectdata field not found or not a string")
	}

	return objectData, nil
}

// GetInfo calls the getinfo RPC method
func (c *Client) GetInfo(ctx context.Context) (*ChainInfo, error) {
	result, err := c.Call(ctx, "getinfo")
	if err != nil {
		return nil, fmt.Errorf("getinfo failed: %w", err)
	}

	var info ChainInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, fmt.Errorf("failed to parse getinfo result: %w", err)
	}

	return &info, nil
}

// ChainInfo represents blockchain information
type ChainInfo struct {
	Name         string `json:"name"`         // Chain name (e.g., "VRSC", "VRSCTEST")
	Blocks       int64  `json:"blocks"`       // Current block height
	Version      int    `json:"version"`      // Daemon version
	Connections  int    `json:"connections"`  // Number of peer connections
	LongestChain int64  `json:"longestchain"` // Longest chain height
	Testnet      bool   `json:"testnet"`      // Whether this is testnet
}

// recordMetrics records call metrics
func (c *Client) recordMetrics(duration time.Duration, err error) {
	c.totalDuration.Add(duration.Microseconds())
	if err != nil {
		c.errorCount.Add(1)
	}
}

// Stats returns client statistics
func (c *Client) Stats() Stats {
	requests := c.requestCount.Load()
	errors := c.errorCount.Load()
	totalDuration := time.Duration(c.totalDuration.Load()) * time.Microsecond

	var avgDuration time.Duration
	if requests > 0 {
		avgDuration = totalDuration / time.Duration(requests)
	}

	var errorRate float64
	if requests > 0 {
		errorRate = float64(errors) / float64(requests)
	}

	return Stats{
		Requests:        requests,
		Errors:          errors,
		TotalDuration:   totalDuration,
		AverageDuration: avgDuration,
		ErrorRate:       errorRate,
	}
}

// Stats contains client statistics
type Stats struct {
	Requests        uint64
	Errors          uint64
	TotalDuration   time.Duration
	AverageDuration time.Duration
	ErrorRate       float64
}

// Close closes the client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
