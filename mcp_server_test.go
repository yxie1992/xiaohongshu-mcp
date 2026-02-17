package main

import "testing"

func TestNormalizeSavedFeedsLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "positive limit", in: 10, want: 10},
		{name: "zero fallback", in: 0, want: 20},
		{name: "negative fallback", in: -1, want: 20},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeSavedFeedsLimit(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeSavedFeedsLimit(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
