package deckgroup

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	if got, err := ValidateName("  Inglês  "); err != nil || got != "Inglês" {
		t.Errorf("ValidateName() = %q, %v; want the trimmed name", got, err)
	}
	if _, err := ValidateName("   "); err == nil {
		t.Error("an empty name should be rejected")
	}
	if _, err := ValidateName(strings.Repeat("a", MaxNameLength+1)); err == nil {
		t.Error("a name over the limit should be rejected")
	}
	// The limit counts characters, not bytes: accents must not eat into it.
	if _, err := ValidateName(strings.Repeat("é", MaxNameLength)); err != nil {
		t.Errorf("60 accented characters should be accepted, got %v", err)
	}
}

func TestNameKey_IgnoresCaseAndSurroundingSpaces(t *testing.T) {
	if NameKey(" Inglês ") != NameKey("inglês") {
		t.Error(`"Inglês" and "inglês" must collide`)
	}
}
