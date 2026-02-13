package graph

import (
	"errors"
	"fmt"
	"strings"
)

// refPrefix is the namespace under which all graph refs are stored.
const refPrefix = "refs/infra/"

// GraphInfo represents a named graph returned by List.
type GraphInfo struct {
	Name string // Graph name (the component after refs/infra/)
}

// ValidateGraphName checks that name is a legal single Git ref-name component
// suitable for use in refs/infra/<name>. Returns nil if valid, or an error
// describing the specific validation failure.
func ValidateGraphName(name string) error {
	if name == "" {
		return errors.New("graph name must not be empty")
	}
	if name == "@" {
		return errors.New("graph name must not be '@'")
	}
	if strings.HasPrefix(name, ".") {
		return errors.New("graph name must not start with '.'")
	}
	if strings.HasSuffix(name, ".") {
		return errors.New("graph name must not end with '.'")
	}
	if strings.HasSuffix(name, ".lock") {
		return errors.New("graph name must not end with '.lock'")
	}
	if strings.Contains(name, "..") {
		return errors.New("graph name must not contain '..'")
	}
	if strings.Contains(name, "@{") {
		return errors.New("graph name must not contain '@{'")
	}
	for _, c := range name {
		if c <= 0x1f || c == 0x7f {
			return fmt.Errorf("graph name must not contain control character 0x%02x", c)
		}
		if strings.ContainsRune(" ~^:?*[\\/", c) {
			return fmt.Errorf("graph name must not contain '%c'", c)
		}
	}
	return nil
}

// graphRefName returns the full ref path for a graph name.
func graphRefName(name string) string {
	return refPrefix + name
}
