package validate

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Errors map[string]string

func (e Errors) HasErrors() bool { return len(e) > 0 }

var (
	emailRE    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_\-]{3,32}$`)
)

func Email(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return errors.New("email is required")
	}
	if len(s) > 254 || !emailRE.MatchString(s) {
		return errors.New("invalid email format")
	}
	return nil
}

func Username(s string) error {
	if !usernameRE.MatchString(strings.TrimSpace(s)) {
		return errors.New("3–32 chars: letters, digits, _, -")
	}
	return nil
}

// Password checks length in runes (not bytes).
func Password(s string) error {
	n := utf8.RuneCountInString(s)
	if n < 12 {
		return errors.New("minimum 12 characters")
	}
	if n > 128 {
		return errors.New("password too long")
	}
	return nil
}

func Required(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " is required")
	}
	return nil
}

func MaxLen(field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return errors.New(field + " is too long (max " + strconv.Itoa(max) + ")")
	}
	return nil
}
