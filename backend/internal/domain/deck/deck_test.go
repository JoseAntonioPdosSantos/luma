package deck

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	got, err := ValidateName("  English  ")
	if err != nil {
		t.Fatalf("ValidateName() returned unexpected error: %v", err)
	}
	if got != "English" {
		t.Errorf("ValidateName() = %q, want %q", got, "English")
	}
}

func TestValidateName_Empty(t *testing.T) {
	if _, err := ValidateName("   "); err == nil {
		t.Error("ValidateName(\"   \") should return an error")
	}
}

func TestValidateName_TooLong(t *testing.T) {
	if _, err := ValidateName(strings.Repeat("a", MaxNameLength+1)); err == nil {
		t.Error("ValidateName() with an over-long name should return an error")
	}
}

func TestValidateDescription_TooLong(t *testing.T) {
	if _, err := ValidateDescription(strings.Repeat("a", MaxDescriptionLength+1)); err == nil {
		t.Error("ValidateDescription() with an over-long description should return an error")
	}
}

func TestValidateDescription_EmptyIsAllowed(t *testing.T) {
	got, err := ValidateDescription("   ")
	if err != nil {
		t.Fatalf("ValidateDescription() returned unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("ValidateDescription() = %q, want empty string", got)
	}
}

func TestIsArchived(t *testing.T) {
	d := Deck{}
	if d.IsArchived() {
		t.Error("a deck with no ArchivedAt should not be archived")
	}
}
