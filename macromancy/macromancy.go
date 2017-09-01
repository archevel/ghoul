package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type Transformer interface {
	Transform(list e.List) e.Expr
}
type Macromancer struct {
	macroGroups []*MacroGroup
}

func NewMacromancer() *Macromancer {

	return &Macromancer{}
}

func (m *Macromancer) Groups() []*MacroGroup {
	return m.macroGroups
}
func (m *Macromancer) Transform(inList e.List) e.Expr {
	return m.transform(inList)
}

func (m *Macromancer) transform(expr e.Expr) e.Expr {
	if l, ok := expr.(e.List); ok && l != e.NIL {
		h := l.Head()
		if sl, ok := subList(l); ok {
			newH := m.expandMacrosAgainst(sl)
			h = newH
			if sl, ok := h.(e.List); ok && e.Identifier("define-syntax").Equiv(sl.Head()) {
				mg, err := NewMacroGroup(sl)
				if err == nil {
					m.macroGroups = append(m.macroGroups, mg)
				}

				if t, ok := tail(l); ok {
					newT := m.expandMacrosAgainst(t)
					return m.transform(newT)
				} else {
					return m.transform(l.Tail())
				}
			}

		}

		h = m.transform(h)

		t := m.transform(l.Tail())

		return e.Cons(h, t)

	} else {
		return expr
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
	t, ok := l.Tail().(e.List)
	return t, ok
}

func subList(l e.List) (e.List, bool) {
	h, ok := l.Head().(e.List)
	return h, ok
}
