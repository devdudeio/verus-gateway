package domain

import (
	"testing"
)

func TestFileRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *FileRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid request",
			req: &FileRequest{
				TXID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID:  "vrsctest",
				Filename: "document.pdf",
			},
			wantErr: false,
		},
		{
			name: "Valid request with EVK",
			req: &FileRequest{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID: "vrsctest",
				EVK:     "zxviews1q0duytgcqqqqpqre26wkl45gvwwwd706xw608hucmvfalr8rgq93rrg27zzp4j7r2rqd8dlsjg7uw7hghtsABCDEF",
			},
			wantErr: false,
		},
		{
			name: "Missing TXID",
			req: &FileRequest{
				ChainID: "vrsctest",
			},
			wantErr: true,
			errMsg:  "txid is required",
		},
		{
			name: "TXID too short",
			req: &FileRequest{
				TXID:    "0123456789abcdef",
				ChainID: "vrsctest",
			},
			wantErr: true,
			errMsg:  "txid must be exactly 64 characters",
		},
		{
			name: "TXID too long",
			req: &FileRequest{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdefextra",
				ChainID: "vrsctest",
			},
			wantErr: true,
			errMsg:  "txid must be exactly 64 characters",
		},
		{
			name: "TXID invalid hex",
			req: &FileRequest{
				TXID:    "0123456789abcdefGHIJKLMNOPQRSTUVWXYZghijklmnopqrstuvwxyz01234567",
				ChainID: "vrsctest",
			},
			wantErr: true,
			errMsg:  "txid must be valid hex",
		},
		{
			name: "Missing ChainID",
			req: &FileRequest{
				TXID: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			wantErr: true,
			errMsg:  "chain_id is required",
		},
		{
			name: "ChainID too long",
			req: &FileRequest{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID: "this_chain_id_is_way_too_long_and_exceeds_the_maximum_allowed_length",
			},
			wantErr: true,
			errMsg:  "chain_id too long",
		},
		{
			name: "ChainID invalid characters",
			req: &FileRequest{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID: "vrsc@test",
			},
			wantErr: true,
			errMsg:  "chain_id contains invalid characters",
		},
		{
			name: "Filename too long",
			req: &FileRequest{
				TXID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID:  "vrsctest",
				Filename: string(make([]byte, 256)),
			},
			wantErr: true,
			errMsg:  "filename too long",
		},
		{
			name: "Filename with path traversal (..))",
			req: &FileRequest{
				TXID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID:  "vrsctest",
				Filename: "../etc/passwd",
			},
			wantErr: true,
			errMsg:  "filename contains invalid path characters",
		},
		{
			name: "Filename with forward slash",
			req: &FileRequest{
				TXID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID:  "vrsctest",
				Filename: "path/to/file.txt",
			},
			wantErr: true,
			errMsg:  "filename contains invalid path characters",
		},
		{
			name: "Filename with backslash",
			req: &FileRequest{
				TXID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID:  "vrsctest",
				Filename: "path\\to\\file.txt",
			},
			wantErr: true,
			errMsg:  "filename contains invalid path characters",
		},
		{
			name: "Filename with special characters",
			req: &FileRequest{
				TXID:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID:  "vrsctest",
				Filename: "file<script>.txt",
			},
			wantErr: true,
			errMsg:  "filename contains invalid characters",
		},
		{
			name: "EVK too short",
			req: &FileRequest{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID: "vrsctest",
				EVK:     "zxviews1q0duytgcqqqqpqre26wkl",
			},
			wantErr: true,
			errMsg:  "viewing key has invalid length",
		},
		{
			name: "EVK invalid format (doesn't start with zxviews)",
			req: &FileRequest{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID: "vrsctest",
				EVK:     "invalid1q0duytgcqqqqpqre26wkl45gvwwwd706xw608hucmvfalr8rgq93rrg27zzp4j7r2rqd8dlsjg7uw7hghtsABCDEF",
			},
			wantErr: true,
			errMsg:  "viewing key has invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FileRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("FileRequest.Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestFileRequest_CacheKey(t *testing.T) {
	tests := []struct {
		name string
		req  *FileRequest
		want string
	}{
		{
			name: "Without EVK",
			req: &FileRequest{
				TXID:    "abc123",
				ChainID: "vrsctest",
			},
			want: "vrsctest:abc123",
		},
		{
			name: "With EVK",
			req: &FileRequest{
				TXID:    "abc123",
				ChainID: "vrsctest",
				EVK:     "zxviews1q0duytgcqqqqpqre26wkl45gvwwwd706xw608hucmvfalr8rgq93rrg27zzp4j7r2rqd8dlsjg7uw7hghts",
			},
			want: "vrsctest:abc123:encrypted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.CacheKey(); got != tt.want {
				t.Errorf("FileRequest.CacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
