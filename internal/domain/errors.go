package domain

import (
	"errors"
	"fmt"
)

// Common sentinel errors
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("not found")

	// ErrInvalidInput indicates invalid input parameters
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = errors.New("unauthorized")

	// ErrRateLimited indicates rate limit exceeded
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrCacheMiss indicates cache miss
	ErrCacheMiss = errors.New("cache miss")

	// ErrChainNotFound indicates chain not found
	ErrChainNotFound = errors.New("chain not found")

	// ErrChainDisabled indicates chain is disabled
	ErrChainDisabled = errors.New("chain is disabled")

	// ErrRPCError indicates RPC call failed
	ErrRPCError = errors.New("rpc error")

	// ErrDecryptionFailed indicates decryption failed
	ErrDecryptionFailed = errors.New("decryption failed")

	// ErrDecompressionFailed indicates decompression failed
	ErrDecompressionFailed = errors.New("decompression failed")

	// ErrInvalidTXID indicates invalid transaction ID
	ErrInvalidTXID = errors.New("invalid txid")

	// ErrInvalidEVK indicates invalid encryption viewing key
	ErrInvalidEVK = errors.New("invalid evk")

	// ErrFileTooLarge indicates file exceeds size limit
	ErrFileTooLarge = errors.New("file too large")

	// ErrInvalidFilename indicates invalid filename
	ErrInvalidFilename = errors.New("invalid filename")

	// ErrUnsupportedFormat indicates unsupported file format
	ErrUnsupportedFormat = errors.New("unsupported format")
)

// Error represents a domain error with context
type Error struct {
	// Code is a machine-readable error code
	Code string

	// Message is a human-readable error message
	Message string

	// Err is the underlying error
	Err error

	// HTTPStatus is the suggested HTTP status code
	HTTPStatus int

	// Details contains additional error context
	Details map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new domain error
func NewError(code, message string, httpStatus int, err error) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		Err:        err,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *Error) WithDetail(key string, value interface{}) *Error {
	e.Details[key] = value
	return e
}

// Common error constructors

// NewNotFoundError creates a not found error
func NewNotFoundError(resource, id string) *Error {
	return NewError(
		"NOT_FOUND",
		fmt.Sprintf("%s not found", resource),
		404,
		ErrNotFound,
	).WithDetail("resource", resource).WithDetail("id", id)
}

// NewInvalidInputError creates an invalid input error
func NewInvalidInputError(field, reason string) *Error {
	return NewError(
		"INVALID_INPUT",
		fmt.Sprintf("invalid input: %s", reason),
		400,
		ErrInvalidInput,
	).WithDetail("field", field).WithDetail("reason", reason)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(limit int, window string) *Error {
	return NewError(
		"RATE_LIMIT_EXCEEDED",
		"rate limit exceeded",
		429,
		ErrRateLimited,
	).WithDetail("limit", limit).WithDetail("window", window)
}

// NewRPCError creates an RPC error
func NewRPCError(method string, err error) *Error {
	return NewError(
		"RPC_ERROR",
		fmt.Sprintf("rpc call failed: %s", method),
		502,
		err,
	).WithDetail("method", method)
}

// NewDecryptionError creates a decryption error
func NewDecryptionError(txid string, err error) *Error {
	return NewError(
		"DECRYPTION_FAILED",
		"failed to decrypt data",
		500,
		err,
	).WithDetail("txid", txid)
}

// NewChainError creates a chain-related error
func NewChainError(chainID, reason string) *Error {
	return NewError(
		"CHAIN_ERROR",
		fmt.Sprintf("chain error: %s", reason),
		400,
		nil,
	).WithDetail("chain_id", chainID).WithDetail("reason", reason)
}

// NewDecompressionError creates a decompression error
func NewDecompressionError(reason string) *Error {
	return NewError(
		"DECOMPRESSION_FAILED",
		fmt.Sprintf("decompression failed: %s", reason),
		500,
		ErrDecompressionFailed,
	).WithDetail("reason", reason)
}
