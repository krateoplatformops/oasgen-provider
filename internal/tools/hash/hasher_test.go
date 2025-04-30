package hasher

import (
	"testing"
)

func BenchmarkHash(b *testing.B) {
	input := []any{"test", 123, true, "another string", 456.78}

	h := NewFNVObjectHash()

	for i := 0; i < b.N; i++ {
		err := h.SumHash(input...)
		if err != nil {
			b.Fatalf("Hash() failed: %v", err)
		}
	}
}

func TestHash(t *testing.T) {
	tests := []struct {
		name    string
		input   []any
		wantErr bool
	}{
		{
			name:    "Single string input",
			input:   []any{"test"},
			wantErr: false,
		},
		{
			name:    "Multiple inputs",
			input:   []any{"test", 123, true},
			wantErr: false,
		},
		{
			name:    "Empty input",
			input:   []any{},
			wantErr: false,
		},
		{
			name:    "Nil input",
			input:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewFNVObjectHash()
			err := h.SumHash(tt.input...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Hash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := h.GetHash()
			if got == "" && !tt.wantErr {
				t.Errorf("Hash() returned empty string, expected valid hash")
			}
		})
	}
}
