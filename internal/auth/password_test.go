package auth

import (
	"strings"
	"testing"
)

func TestPasswordHashAndVerify(t *testing.T) {
	password := "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("hash prefix: %q", hash[:20])
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil || !ok {
		t.Fatalf("verify failed: %v", err)
	}

	ok, _ = VerifyPassword("wrong", hash)
	if ok {
		t.Fatal("verify should fail for wrong password")
	}
}

func TestVerifyRejectsHugeParams(t *testing.T) {
	// DoS protection: a substituted hash with huge memory must not be executed.
	evil := "$argon2id$v=19$m=1048576,t=1,p=1$c2FsdA$aGFzaA"
	if _, err := VerifyPassword("x", evil); err == nil {
		t.Fatal("expected error for out-of-range params")
	}
}
