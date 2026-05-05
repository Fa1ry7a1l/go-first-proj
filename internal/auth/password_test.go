package auth

import "testing"

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !CheckPassword(hash, "secret") {
		t.Fatal("CheckPassword returned false for valid password")
	}
	if CheckPassword(hash, "other") {
		t.Fatal("CheckPassword returned true for invalid password")
	}
}
