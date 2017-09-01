package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type Transformer interface {
	Transform(list e.List) (e.Expr, error)
}
type Macromancer struct {
	macroGroups []*MacroGroup
}

func NewMacromancer() *Macromancer {

	return &Macromancer{}
}

func (m Macromancer) Transform(inList e.List) (e.Expr, error) {
	return m.transform(inList)
}

func (m *Macromancer) transform(expr e.Expr) (e.Expr, error) {
	if l, ok := expr.(e.List); ok && l != e.NIL {
		h := l.Head()
		if sl, ok := subList(l); ok {
			newH, err := m.expandMacrosAgainst(sl)
			if err != nil {
				return nil, err
			}
			h = newH
			if sl, ok := h.(e.List); ok && e.Identifier("define-syntax").Equiv(sl.Head()) {
				mg, err := NewMacroGroup(sl)
				if err == nil {
					m.macroGroups = append(m.macroGroups, mg)
				}

				if t, ok := tail(l); ok {
					newT, err := m.expandMacrosAgainst(t)
					if err != nil {
						return nil, err
					}
					return m.transform(newT)
				} else {
					return m.transform(sl.Tail())
				}
			}

		}

		h, err := m.transform(h)
		if err != nil {
			return nil, err
		}
		t, err := m.transform(l.Tail())
		if err != nil {
			return nil, err
		}
		return &e.Pair{h, t}, nil

	} else {
		return expr, nil
	}
}

func (m *Macromancer) expandMacrosAgainst(subList e.List) (e.Expr, error) {
	var subExpr e.Expr = subList
	for _, mg := range m.macroGroups {
		macros := mg.Matches(subList)
		if macros != nil {
			for _, macro := range macros {
				if ok, bound := macro.Matches(subList); ok {
					newSubExpr, err := macro.Expand(bound)
					if err != nil {
						return nil, err
					}
					return newSubExpr, nil
				}
			}
			break
		}
	}
	return subExpr, nil
}

func tail(l e.List) (e.List, bool) {
	t, ok := l.Tail().(e.List)
	return t, ok
}

func subList(l e.List) (e.List, bool) {
	h, ok := l.Head().(e.List)
	return h, ok
}
