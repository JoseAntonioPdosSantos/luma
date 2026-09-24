package user

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	got := NormalizeEmail("  Jose.Antonio@Example.COM  ")
	want := "jose.antonio@example.com"
	if got != want {
		t.Errorf("NormalizeEmail() = %q, want %q", got, want)
	}
}

func TestValidateEmail_Valid(t *testing.T) {
	got, err := ValidateEmail("  User@Example.com ")
	if err != nil {
		t.Fatalf("ValidateEmail() returned unexpected error: %v", err)
	}
	if got != "user@example.com" {
		t.Errorf("ValidateEmail() = %q, want %q", got, "user@example.com")
	}
}

func TestValidateEmail_Empty(t *testing.T) {
	if _, err := ValidateEmail("   "); err == nil {
		t.Error("ValidateEmail(\"   \") should return an error")
	}
}

func TestValidateEmail_Malformed(t *testing.T) {
	if _, err := ValidateEmail("not-an-email"); err == nil {
		t.Error("ValidateEmail(\"not-an-email\") should return an error")
	}
}

func TestValidateEmail_TooLong(t *testing.T) {
	longLocal := strings.Repeat("a", MaxEmailLength)
	if _, err := ValidateEmail(longLocal + "@example.com"); err == nil {
		t.Error("ValidateEmail() with an over-long address should return an error")
	}
}

func TestValidateLanguage_Valid(t *testing.T) {
	for _, lang := range []string{LanguagePortuguese, LanguageEnglish, LanguageSpanish, LanguageFrench} {
		got, err := ValidateLanguage(lang)
		if err != nil {
			t.Fatalf("ValidateLanguage(%q) returned unexpected error: %v", lang, err)
		}
		if got != lang {
			t.Errorf("ValidateLanguage(%q) = %q, want %q", lang, got, lang)
		}
	}
}

func TestValidateLanguage_Invalid(t *testing.T) {
	for _, lang := range []string{"", "de", "PT", "pt-BR"} {
		if _, err := ValidateLanguage(lang); err == nil {
			t.Errorf("ValidateLanguage(%q) should return an error", lang)
		}
	}
}
