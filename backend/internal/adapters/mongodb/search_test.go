package mongodb

import "testing"

func TestSearchPattern(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"rescue", "r[eèéêëEÈÉÊË]s[cçCÇ][uùúûüUÙÚÛÜ][eèéêëEÈÉÊË]"},
		{"a.b", "[aàáâãäåAÀÁÂÃÄÅ]\\.b"},
		{"  x  ", "x"},
		{"(1+1)", "\\(1\\+1\\)"},
	}
	for _, tt := range tests {
		if got := searchPattern(tt.query); got != tt.want {
			t.Errorf("searchPattern(%q) = %q, want %q", tt.query, got, tt.want)
		}
	}
}
