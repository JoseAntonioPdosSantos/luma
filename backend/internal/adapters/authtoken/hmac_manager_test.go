package authtoken

import (
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestHMACManager_IssueThenVerify(t *testing.T) {
	manager := NewHMACManager("test-secret", time.Hour)

	token, expiresAt, err := manager.Issue("user-123", fixedNow)
	if err != nil {
		t.Fatalf("Issue() error: %v", err)
	}
	if !expiresAt.Equal(fixedNow.Add(time.Hour)) {
		t.Errorf("expiresAt = %v, want %v", expiresAt, fixedNow.Add(time.Hour))
	}

	userID, err := manager.Verify(token, fixedNow)
	if err != nil {
		t.Fatalf("Verify() error: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("Verify() userID = %q, want %q", userID, "user-123")
	}
}

func TestHMACManager_Verify_Expired(t *testing.T) {
	manager := NewHMACManager("test-secret", time.Hour)

	token, _, _ := manager.Issue("user-123", fixedNow)

	_, err := manager.Verify(token, fixedNow.Add(2*time.Hour))
	if err == nil {
		t.Fatal("Verify() should reject an expired token")
	}
}

func TestHMACManager_Verify_TamperedSignature(t *testing.T) {
	manager := NewHMACManager("test-secret", time.Hour)

	token, _, _ := manager.Issue("user-123", fixedNow)
	tampered := token[:len(token)-1] + "x"

	_, err := manager.Verify(tampered, fixedNow)
	if err == nil {
		t.Fatal("Verify() should reject a token with a tampered signature")
	}
}

func TestHMACManager_Verify_WrongSecret(t *testing.T) {
	issuer := NewHMACManager("secret-a", time.Hour)
	verifier := NewHMACManager("secret-b", time.Hour)

	token, _, _ := issuer.Issue("user-123", fixedNow)

	_, err := verifier.Verify(token, fixedNow)
	if err == nil {
		t.Fatal("Verify() should reject a token signed with a different secret")
	}
}

func TestHMACManager_Verify_Malformed(t *testing.T) {
	manager := NewHMACManager("test-secret", time.Hour)

	if _, err := manager.Verify("not-a-valid-token", fixedNow); err == nil {
		t.Fatal("Verify() should reject a malformed token")
	}
}

func TestNewHMACManager_DefaultTTL(t *testing.T) {
	manager := NewHMACManager("test-secret", 0)
	_, expiresAt, _ := manager.Issue("user-123", fixedNow)
	if !expiresAt.Equal(fixedNow.Add(defaultTTL)) {
		t.Errorf("expiresAt = %v, want %v", expiresAt, fixedNow.Add(defaultTTL))
	}
}
