package studyprofile

import (
	"strings"
	"testing"
)

func TestDefault_IsTheBuiltInConfiguration(t *testing.T) {
	d := Default()
	if !d.IsDefault() || d.ID != DefaultID || d.Name != DefaultName || d.DailyCardLimit != nil {
		t.Errorf("Default() = %+v, want the built-in one without a daily goal", d)
	}
	if err := d.Rules.Validate(); err != nil {
		t.Errorf("the default rules must be valid: %v", err)
	}
}

func TestValidateName(t *testing.T) {
	if got, err := ValidateName("  Prova de inglês  "); err != nil || got != "Prova de inglês" {
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
	if NameKey(" Prova ") != NameKey("prova") {
		t.Error(`"Prova" and "prova" must collide`)
	}
}

func TestValidateDailyCardLimit(t *testing.T) {
	ok, low, high := 30, 0, MaxDailyCardLimit+1
	if err := ValidateDailyCardLimit(nil); err != nil {
		t.Errorf("nil (no goal) must be valid: %v", err)
	}
	if err := ValidateDailyCardLimit(&ok); err != nil {
		t.Errorf("30 must be valid: %v", err)
	}
	if ValidateDailyCardLimit(&low) == nil || ValidateDailyCardLimit(&high) == nil {
		t.Error("0 and 201 must be rejected")
	}
}
