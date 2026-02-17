package main

import "testing"

func TestParsePositiveLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		limitParam   string
		defaultLimit int
		wantLimit    int
		wantErr      bool
	}{
		{
			name:         "empty uses default",
			limitParam:   "",
			defaultLimit: 20,
			wantLimit:    20,
			wantErr:      false,
		},
		{
			name:         "valid positive integer",
			limitParam:   "50",
			defaultLimit: 20,
			wantLimit:    50,
			wantErr:      false,
		},
		{
			name:         "non integer returns error",
			limitParam:   "abc",
			defaultLimit: 20,
			wantLimit:    0,
			wantErr:      true,
		},
		{
			name:         "zero returns error",
			limitParam:   "0",
			defaultLimit: 20,
			wantLimit:    0,
			wantErr:      true,
		},
		{
			name:         "negative returns error",
			limitParam:   "-3",
			defaultLimit: 20,
			wantLimit:    0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePositiveLimit(tt.limitParam, tt.defaultLimit)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePositiveLimit(%q, %d) expected error, got nil", tt.limitParam, tt.defaultLimit)
				}
				return
			}

			if err != nil {
				t.Fatalf("parsePositiveLimit(%q, %d) unexpected error: %v", tt.limitParam, tt.defaultLimit, err)
			}
			if got != tt.wantLimit {
				t.Fatalf("parsePositiveLimit(%q, %d) = %d, want %d", tt.limitParam, tt.defaultLimit, got, tt.wantLimit)
			}
		})
	}
}
