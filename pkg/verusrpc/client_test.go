package verusrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		URL:      "http://localhost:27486",
		User:     "user",
		Password: "pass",
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.url != cfg.URL {
		t.Errorf("expected url %s, got %s", cfg.URL, client.url)
	}

	if client.timeout == 0 {
		t.Error("expected default timeout to be set")
	}

	if client.maxRetries == 0 {
		t.Error("expected default maxRetries to be set")
	}
}

func TestClient_Call_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			t.Error("invalid basic auth")
		}

		// Return success response
		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"test result"`),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:      server.URL,
		User:     "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	})

	ctx := context.Background()
	result, err := client.Call(ctx, "testmethod", "param1", "param2")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resultStr string
	if err := json.Unmarshal(result, &resultStr); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if resultStr != "test result" {
		t.Errorf("expected 'test result', got '%s'", resultStr)
	}
}

func TestClient_Call_RPCError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:      server.URL,
		User:     "user",
		Password: "pass",
		Timeout:  5 * time.Second,
	})

	ctx := context.Background()
	_, err := client.Call(ctx, "nonexistent")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Check if error contains RPCError (may be wrapped)
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestClient_Call_HTTPError(t *testing.T) {
	// Create mock server that returns HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:        server.URL,
		User:       "user",
		Password:   "pass",
		Timeout:    5 * time.Second,
		MaxRetries: 0, // Disable retries for this test
	})

	ctx := context.Background()
	_, err := client.Call(ctx, "testmethod")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_Call_ContextCanceled(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:      server.URL,
		User:     "user",
		Password: "pass",
		Timeout:  5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Call(ctx, "testmethod")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if ctx.Err() == nil {
		t.Error("expected context to be canceled")
	}
}

func TestClient_DecryptData(t *testing.T) {
	expectedResult := "48656c6c6f" // hex for "Hello"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// DecryptData expects an array of objects with objectdata field
		resultArray := []map[string]interface{}{
			{"objectdata": expectedResult},
		}
		resultJSON, _ := json.Marshal(resultArray)

		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  resultJSON,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:      server.URL,
		User:     "user",
		Password: "pass",
	})

	ctx := context.Background()
	result, err := client.DecryptData(ctx, "txid123", "evk456")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("expected '%s', got '%s'", expectedResult, result)
	}
}

func TestClient_GetInfo(t *testing.T) {
	expectedInfo := ChainInfo{
		Name:        "VRSC",
		Blocks:      12345,
		Version:     1000000,
		Connections: 8,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resultJSON, _ := json.Marshal(expectedInfo)
		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  resultJSON,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:      server.URL,
		User:     "user",
		Password: "pass",
	})

	ctx := context.Background()
	info, err := client.GetInfo(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Name != expectedInfo.Name {
		t.Errorf("expected name %s, got %s", expectedInfo.Name, info.Name)
	}

	if info.Blocks != expectedInfo.Blocks {
		t.Errorf("expected blocks %d, got %d", expectedInfo.Blocks, info.Blocks)
	}
}

func TestClient_Stats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"result"`),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:      server.URL,
		User:     "user",
		Password: "pass",
	})

	ctx := context.Background()

	// Make some calls
	client.Call(ctx, "method1")
	client.Call(ctx, "method2")

	stats := client.Stats()

	if stats.Requests != 2 {
		t.Errorf("expected 2 requests, got %d", stats.Requests)
	}

	if stats.AverageDuration == 0 {
		t.Error("expected non-zero average duration")
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++

		// Fail first 2 attempts, succeed on 3rd
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := Response{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"success"`),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		URL:        server.URL,
		User:       "user",
		Password:   "pass",
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	})

	ctx := context.Background()
	result, err := client.Call(ctx, "testmethod")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	var resultStr string
	json.Unmarshal(result, &resultStr)
	if resultStr != "success" {
		t.Errorf("expected 'success', got '%s'", resultStr)
	}
}
