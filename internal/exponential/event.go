package exponential

import (
	"fmt"
	"regexp"
)

// ValidateUser validates that a user string matches the expected format "Name <email>".
func ValidateUser(user string) error {
	// Pattern: "One or more chars" followed by space(s), then "<email@domain>"
	re := regexp.MustCompile(`^.+\s+<[^<>]+@[^<>]+>$`)
	if !re.MatchString(user) {
		return fmt.Errorf("invalid user format: expected 'Name <email>', got %q", user)
	}
	return nil
}
