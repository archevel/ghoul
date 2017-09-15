package evaluator

import e "github.com/archevel/ghoul/expressions"

type Function struct {
	Fun *func(e.List, *Evaluator) (e.Expr, error)
}

func (e Function) Repr() string {
	return "#<procedure>"
}

func (e Function) Equiv(expr e.Expr) bool {
	switch v := expr.(type) {
	case Function:
		return e == v
	case *Function:
		return e == *v
	}

	return false
}
