package utils

import (
	"fmt"
	"strings"
)

// return the pieces of id `a:b` as a, b
func ParseTwoPartID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected ID format (%q). Expected <part1>:<part2>", id)
	}

	return parts[0], parts[1], nil
}

// format the strings into an id `a:b`
func BuildTwoPartID(a, b *string) string {
	return fmt.Sprintf("%s:%s", *a, *b)
}

// return the pieces of id `a:b:c` as a, b, c
func ParseThreePartID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("Unexpected ID format (%q). Expected <part1>:<part2>:<part3>", id)
	}

	return parts[0], parts[1], parts[2], nil
}

// format the strings into an id `a:b:c`
func BuildThreePartID(a, b, c *string) string {
	return fmt.Sprintf("%s:%s:%s", *a, *b, *c)
}
