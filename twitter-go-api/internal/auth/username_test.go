package auth

import (
	"regexp"
	"testing"
)

// Must match the users.username CHECK constraint, or sign-up fails.
var usernameConstraint = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)

func TestGeneratedUsernameSatisfiesConstraint(t *testing.T) {
	for _, email := range []string{
		"alice@example.com",
		"a.very.long.local.part.indeed@example.com",
		"ünïcødé+tag@example.com",
		"@example.com",
		"x@y",
	} {
		if got := generatedUsername(email); !usernameConstraint.MatchString(got) {
			t.Fatalf("generatedUsername(%q) = %q violates the users.username constraint", email, got)
		}
	}
}
