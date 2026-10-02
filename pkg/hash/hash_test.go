package hash

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordByteBoundaries(t *testing.T) {
	for _, password := range []string{"", "1234567", strings.Repeat("a", 73), strings.Repeat("密", 25)} {
		if err := ValidatePassword(password); err == nil {
			t.Fatalf("invalid password with %d bytes was accepted", len(password))
		}
	}
	for _, password := range []string{"12345678", strings.Repeat("a", 72), strings.Repeat("密", 24)} {
		if err := ValidatePassword(password); err != nil {
			t.Fatalf("valid password with %d bytes was rejected: %v", len(password), err)
		}
	}
}

func TestBcryptHashReturnsFailure(t *testing.T) {
	encoded, err := BcryptHash(strings.Repeat("a", 73))
	if encoded != "" || !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Fatalf("bcrypt failure was swallowed: hash=%q err=%v", encoded, err)
	}
	password := strings.Repeat("a", 72)
	encoded, err = BcryptHash(password)
	if err != nil || !BcryptCheck(password, encoded) || BcryptCheck("wrong", encoded) {
		t.Fatal("bcrypt boundary password did not round-trip", err)
	}
}
