package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// OWASP 2024 for argon2id: m=64 MiB, t=3, p=4.
var DefaultParams = &Argon2Params{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// Hard limits for parameters read from DB (DoS protection):
// if an attacker substitutes a hash with m=1GB, argon2 would eat all memory.
const (
	maxMemory      = 256 * 1024 // 256 MiB
	maxIterations  = 10
	maxParallelism = 16
	maxKeyLength   = 64
)

func HashPassword(password string) (string, error) {
	p := DefaultParams
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		// crypto/rand on supported platforms does not fail, but errcheck
		// in CI does not like ignoring the error. Keep the check.
		return "", fmt.Errorf("rand: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	if version != argon2.Version {
		return false, errors.New("incompatible argon2 version")
	}

	p := &Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return false, err
	}
	if p.Memory == 0 || p.Memory > maxMemory ||
		p.Iterations == 0 || p.Iterations > maxIterations ||
		p.Parallelism == 0 || p.Parallelism > maxParallelism {
		return false, errors.New("argon2 params out of range")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	if len(expected) == 0 || len(expected) > maxKeyLength {
		return false, errors.New("invalid key length")
	}
	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(expected))

	actual := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
