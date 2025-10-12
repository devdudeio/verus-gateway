package crypto

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

// Mock RPC client for testing
type mockRPCClient struct {
	hexData string
	err     error
}

func (m *mockRPCClient) DecryptData(ctx context.Context, txid, evk string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.hexData, nil
}

func TestNewDecryptor(t *testing.T) {
	client := &mockRPCClient{}
	d := NewDecryptor(client)

	if d == nil {
		t.Fatal("expected non-nil decryptor")
	}
}

func TestValidateTXID(t *testing.T) {
	tests := []struct {
		name    string
		txid    string
		wantErr bool
	}{
		{
			name:    "valid txid",
			txid:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "uppercase valid",
			txid:    "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF",
			wantErr: false,
		},
		{
			name:    "too short",
			txid:    "1234567890abcdef",
			wantErr: true,
		},
		{
			name:    "too long",
			txid:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef00",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			txid:    "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: true,
		},
		{
			name:    "empty",
			txid:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTXID(tt.txid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTXID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidTXID) {
				t.Errorf("expected ErrInvalidTXID, got %v", err)
			}
		})
	}
}

func TestValidateEVK(t *testing.T) {
	tests := []struct {
		name    string
		evk     string
		wantErr bool
	}{
		{
			name:    "valid evk",
			evk:     "zxviews1234567890abcdefghijklmnopqrstuvwxyz",
			wantErr: false,
		},
		{
			name:    "missing prefix",
			evk:     "1234567890abcdefghijklmnopqrstuvwxyz",
			wantErr: true,
		},
		{
			name:    "uppercase letters",
			evk:     "zxviews1234567890ABCDEF",
			wantErr: true,
		},
		{
			name:    "empty",
			evk:     "",
			wantErr: true,
		},
		{
			name:    "only prefix",
			evk:     "zxviews",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEVK(tt.evk)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEVK() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, domain.ErrInvalidEVK) {
				t.Errorf("expected ErrInvalidEVK, got %v", err)
			}
		})
	}
}

func TestDecryptor_DecryptData(t *testing.T) {
	validTXID := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	validEVK := "zxviews1234567890abcdefghijklmnopqrstuvwxyz"

	t.Run("successful decryption", func(t *testing.T) {
		expectedData := []byte("Hello World")
		hexData := hex.EncodeToString(expectedData)

		mockClient := &mockRPCClient{
			hexData: hexData,
		}

		d := NewDecryptor(mockClient)
		result, err := d.DecryptData(context.Background(), validTXID, validEVK)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if string(result) != string(expectedData) {
			t.Errorf("expected %q, got %q", string(expectedData), string(result))
		}
	})

	t.Run("invalid txid", func(t *testing.T) {
		d := NewDecryptor(&mockRPCClient{})
		_, err := d.DecryptData(context.Background(), "invalid", validEVK)
		if err == nil {
			t.Fatal("expected error for invalid txid")
		}
	})

	t.Run("invalid evk", func(t *testing.T) {
		d := NewDecryptor(&mockRPCClient{})
		_, err := d.DecryptData(context.Background(), validTXID, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid evk")
		}
	})

	t.Run("rpc error", func(t *testing.T) {
		mockClient := &mockRPCClient{
			err: errors.New("rpc connection failed"),
		}

		d := NewDecryptor(mockClient)
		_, err := d.DecryptData(context.Background(), validTXID, validEVK)
		if err == nil {
			t.Fatal("expected error from RPC failure")
		}
	})

	t.Run("invalid hex data", func(t *testing.T) {
		mockClient := &mockRPCClient{
			hexData: "invalid hex data",
		}

		d := NewDecryptor(mockClient)
		_, err := d.DecryptData(context.Background(), validTXID, validEVK)
		if err == nil {
			t.Fatal("expected error for invalid hex data")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockClient := &mockRPCClient{
			err: context.Canceled,
		}

		d := NewDecryptor(mockClient)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := d.DecryptData(ctx, validTXID, validEVK)
		if err == nil {
			t.Fatal("expected error from canceled context")
		}
	})
}
