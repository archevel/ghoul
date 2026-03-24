package mummy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

// Mummy wraps a Go value with type metadata, used by wraith-generated
// sarcophagi. For ad-hoc wrapping without type names, see expressions.Foreign.
type Mummy struct {
	wrapped  any
	typeName string
}

func Entomb(val any, typeName string) *Mummy {
	return &Mummy{wrapped: val, typeName: typeName}
}

func (m *Mummy) Unwrap() any {
	return m.wrapped
}

func (m *Mummy) Repr() string {
	return fmt.Sprintf("#<mummy:%s>", m.typeName)
}

// Equiv compares wrapped values using ==. Go panics when comparing
// uncomparable types (slices, maps, functions) with ==, so we recover
// and return false — those types have no meaningful equality.
func (m *Mummy) Equiv(other e.Expr) bool {
	o, ok := other.(*Mummy)
	if !ok {
		return false
	}
	defer func() { recover() }()
	return m.wrapped == o.wrapped
}
