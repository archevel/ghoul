package expressions

import (
	"bytes"
	"fmt"
	"strconv"
)

type Expr interface {
	Repr() string
	Equiv(Expr) bool
}

type Boolean bool

func (e Boolean) Repr() string {
	if e {
		return "#t"
	} else {
		return "#f"
	}
}

func (e Boolean) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Boolean:
		return v == e
	default:
		return false
	}

}

type Integer int64

func (e Integer) Repr() string {
	return fmt.Sprintf("%d", e)
}

func (e Integer) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Integer:
		return v == e
	case Float:
		return v == Float(e)
	default:
		return false
	}
}

type Float float64

func (e Float) Repr() string {
	return strconv.FormatFloat(float64(e), 'g', -1, 64)
}

func (e Float) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Integer:
		return Float(v) == e
	case Float:
		return v == e
	default:
		return false
	}
}

type String string

func (e String) Repr() string {
	return fmt.Sprintf(`"%s"`, e)
}

func (e String) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case String:
		return v == e
	default:
		return false
	}
}

type Function struct {
	Fun *func(args List) (Expr, error)
}

func (e Function) Repr() string {
	return "#<procedure>"
}

func (e Function) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Function:
		return e == v
	case *Function:
		return e == *v
	}

	return false
}

type Quote struct {
	Quoted Expr
}

func (e Quote) Repr() string {
	return fmt.Sprintf("'%s", e.Quoted.Repr())
}

func (e Quote) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Quote:
		return e.Quoted.Equiv(v.Quoted)
	case *Quote:
		return e.Quoted.Equiv(v.Quoted)
	default:
		return false
	}
	return true
}

type Identifier string

func (e Identifier) Repr() string { return string(e) }

func (e Identifier) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Identifier:
		return v == e
	default:
		return false
	}
}

type List interface {
	Head() Expr
	Tail() Expr
	Expr
}

type Pair struct {
	H Expr
	T Expr
}

func (p Pair) Head() Expr {
	return p.H
}

func (p Pair) Tail() Expr {
	return p.T
}

func (p Pair) Repr() string {

	var buffer bytes.Buffer
	buffer.WriteRune('(')
	var l List = p

	for l != NIL {
		buffer.WriteString(l.Head().Repr())
		t := l.Tail()

		if t != NIL {
			buffer.WriteRune(' ')
		}
		var ok bool
		l, ok = t.(List)
		if !ok {
			buffer.WriteRune('.')
			buffer.WriteRune(' ')
			buffer.WriteString(t.Repr())
			l = NIL
		}
	}

	buffer.WriteRune(')')
	return buffer.String()
}
func (p Pair) Equiv(expr Expr) bool {
	switch v := expr.(type) {
	case Pair:
		return pairEquiv(p, v)
	case *Pair:
		return pairEquiv(p, v)
	default:
		return false
	}
}

func pairEquiv(x List, y List) bool {
	var xTmp List
	var yTmp List
	eq := true
	xOk, yOk := true, true
	for eq && xOk && yOk && (x != y) {
		eq = x.Head().Equiv(y.Head())

		xTmp, xOk = x.Tail().(List)
		yTmp, yOk = y.Tail().(List)
		if xOk && yOk {
			x = xTmp
			y = yTmp
		}
	}

	if eq && (x != y) {
		eq = x.Tail().Equiv(y.Tail())
	}

	return eq
}

type nilList bool

const NIL nilList = false

func (e nilList) Head() Expr {
	return NIL
}

func (e nilList) Tail() Expr {
	return NIL
}

func (e nilList) Repr() string {
	return "()"
}

func (e nilList) Equiv(expr Expr) bool {
	return expr == NIL
}

func (e nilList) String() string {
	return e.Repr()
}
