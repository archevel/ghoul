package mummy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
)

// Mummy wraps a Go value, preserving it in its sarcophagus for use in Ghoul.
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

func (m *Mummy) Equiv(other e.Expr) bool {
	o, ok := other.(*Mummy)
	if !ok {
		return false
	}
	defer func() { recover() }()
	return m.wrapped == o.wrapped
}
