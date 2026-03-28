package testpkg

import (
	"io"
	"os"
)

// --- Missing import: function uses types from other packages ---

// ReadAll reads everything from a reader (uses io.Reader in signature)
func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// OpenFile opens a file (uses *os.File in return, os.FileMode in param)
func OpenFile(name string, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY, perm)
}

// --- Multi-return: function returns more than one non-error value ---

// SplitNameAge splits "name:age" and returns both
func SplitNameAge(s string) (string, int) {
	for i, c := range s {
		if c == ':' {
			age := 0
			for _, d := range s[i+1:] {
				age = age*10 + int(d-'0')
			}
			return s[:i], age
		}
	}
	return s, 0
}

// --- Syntax: slice parameter types ---

// SumFloats sums a slice of float64 (tests []float64 in signature)
func SumFloats(fs []float64) float64 {
	var total float64
	for _, f := range fs {
		total += f
	}
	return total
}

// ConcatStrings concatenates string slices
func ConcatStrings(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}

// --- Overflow: uint constants ---

const MaxUnsigned uint = ^uint(0)

// --- Variable redeclaration: parameter named 'err' clashes with error return ---

// CloseWithMessage closes something with an error message, returning an error.
// The parameter 'err' clashes with the error return variable.
func CloseWithMessage(err error) error {
	if err != nil {
		return err
	}
	return nil
}

// --- Package name shadowing: parameter named same as package ---

// Exported interface using io types
type ReadCloser interface {
	Read(p []byte) (int, error)
	Close() error
}
