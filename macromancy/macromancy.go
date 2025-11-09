package macromancy

import (
	"fmt"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/logging"
)

type Transformer interface {
	Transform(list e.List) (e.Expr, error)
}
type Macromancer struct {
	logger      logging.Logger
	macroGroups []*MacroGroup
}

func NewMacromancer(logger logging.Logger) *Macromancer {
	return &Macromancer{logger, []*MacroGroup{}}
}

func (m *Macromancer) Groups() []*MacroGroup {
	return m.macroGroups
}
func (m *Macromancer) Transform(inList e.List) (e.Expr, error) {
	return m.transform(inList)
}

func (m *Macromancer) transform(expr e.Expr) (e.Expr, error) {
	if l, ok := expr.(e.List); ok && l != e.NIL {
		h := l.First()
		if sl, ok := subList(l); ok {
			newH := m.expandMacrosAgainst(sl)
			h = newH
			if sl, ok := h.(e.List); ok && e.Identifier("define-syntax").Equiv(sl.First()) {
				mg, err := NewMacroGroup(sl)
				if err != nil {
					return nil, fmt.Errorf("failed to extract macros for syntax definition: %w", err)
				}
				m.macroGroups = append(m.macroGroups, mg)

				if t, ok := tail(l); ok {
					newT := m.expandMacrosAgainst(t)
					return m.transform(newT)
				} else {
					return m.transform(l.Second())
				}
			}

		}

		h, err := m.transform(h)
		if err != nil {
			return nil, err
		}

		t, err := m.transform(l.Second())
		if err != nil {
			return nil, err
		}

		return e.Cons(h, t), nil

	} else {
		return expr, nil
	}
}

func (m *Macromancer) expandMacrosAgainst(subList e.List) e.Expr {
	var subExpr e.Expr = subList
	for _, mg := range m.macroGroups {
		macros := mg.Matches(subList)
		if macros != nil {
			for _, macro := range macros {
				if ok, bound := macro.Matches(subList); ok {
					return macro.Expand(bound)
				}
			}
			break
		}
	}
	return subExpr
}

func tail(l e.List) (e.List, bool) {
	t, ok := l.Tail()
	return t, ok
}

func subList(l e.List) (e.List, bool) {
	h, ok := l.First().(e.List)
	return h, ok
}
