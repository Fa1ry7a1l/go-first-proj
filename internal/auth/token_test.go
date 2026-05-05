package auth

import "testing"

func TestTokenManagerIssueAndVerify(t *testing.T) {
	manager := NewTokenManager("secret")

	token := manager.Issue(42)
	userID, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if userID != 42 {
		t.Fatalf("userID = %d, want 42", userID)
	}
}

func TestTokenManagerRejectsInvalidToken(t *testing.T) {
	manager := NewTokenManager("secret")

	if _, err := manager.Verify("broken"); err != ErrInvalidToken {
		t.Fatalf("Verify error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestTokenManagerRejectsWrongSignature(t *testing.T) {
	issuer := NewTokenManager("secret")
	verifier := NewTokenManager("another-secret")

	if _, err := verifier.Verify(issuer.Issue(42)); err != ErrInvalidToken {
		t.Fatalf("Verify error = %v, want %v", err, ErrInvalidToken)
	}
}
