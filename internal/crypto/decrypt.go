package crypto

import (
	"context"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

var (
	// Validation patterns
	reTXID = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	reEVK  = regexp.MustCompile(`^zxviews[0-9a-z]+$`)
)

// RPCClient interface for calling decryptdata RPC method
type RPCClient interface {
	DecryptData(ctx context.Context, txid, evk string) (string, error)
}

// Decryptor handles decryption of Verus blockchain data
type Decryptor struct {
	client RPCClient
}

// NewDecryptor creates a new decryptor
func NewDecryptor(client RPCClient) *Decryptor {
	return &Decryptor{
		client: client,
	}
}

// DecryptData decrypts data from a transaction using the EVK
func (d *Decryptor) DecryptData(ctx context.Context, txid, evk string) ([]byte, error) {
	// Validate inputs
	if err := ValidateTXID(txid); err != nil {
		return nil, domain.NewInvalidInputError("txid", err.Error())
	}
	if err := ValidateEVK(evk); err != nil {
		return nil, domain.NewInvalidInputError("evk", err.Error())
	}

	// Call RPC client's DecryptData method which returns hex-encoded data
	hexData, err := d.client.DecryptData(ctx, txid, evk)
	if err != nil {
		return nil, domain.NewDecryptionError(txid, err)
	}

	// Decode the hex-encoded data
	data, err := hex.DecodeString(hexData)
	if err != nil {
		return nil, domain.NewDecryptionError(txid, fmt.Errorf("failed to decode hex data: %w", err))
	}

	return data, nil
}

// ValidateTXID validates a transaction ID format
func ValidateTXID(txid string) error {
	if !reTXID.MatchString(txid) {
		return domain.ErrInvalidTXID
	}
	return nil
}

// ValidateEVK validates an encryption viewing key format
func ValidateEVK(evk string) error {
	if !reEVK.MatchString(evk) {
		return domain.ErrInvalidEVK
	}
	return nil
}
