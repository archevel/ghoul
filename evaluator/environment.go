package evaluator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	e "github.com/archevel/ghoul/expressions"
)

// scopeKey must be comparable for use as a map key, so marks are
// encoded as a canonical sorted string rather than a map.
type scopeKey struct {
	Name     string
	MarksKey string
}

func keyFromIdentifier(id e.Identifier) scopeKey {
	return scopeKey{Name: string(id), MarksKey: ""}
}

func keyFromScopedIdentifier(si e.ScopedIdentifier) scopeKey {
	return scopeKey{Name: string(si.Name), MarksKey: canonicalMarks(si.Marks)}
}

func keyFromExpr(variable e.Expr) (scopeKey, bool) {
	switch v := variable.(type) {
	case e.Identifier:
		return keyFromIdentifier(v), true
	case e.ScopedIdentifier:
		return keyFromScopedIdentifier(v), true
	default:
		return scopeKey{}, false
	}
}

func canonicalMarks(marks map[uint64]bool) string {
	if len(marks) == 0 {
		return ""
	}
	ids := make([]uint64, 0, len(marks))
	for k := range marks {
		ids = append(ids, k)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatUint(id, 10)
	}
	return strings.Join(parts, ",")
}

type scope map[scopeKey]e.Expr

type Environment = environment
type environment []*scope

type Registrator interface {
	Register(string, func(e.List) (e.Expr, error))
}

func NewEnvironment() *environment {
	return newEnvWithEmptyScope(&environment{})
}

func (env environment) Register(name string, f func(e.List, *Evaluator) (e.Expr, error)) {
	bindFuncAtBottomAs(e.Identifier(name), Function{&f}, &env)
}

func bindFuncAtBottomAs(id e.Identifier, fun Function, env *environment) {
	scope := bottomScope(env)
	(*scope)[keyFromIdentifier(id)] = fun
}

func RegisterFuncAs(name string, f func(e.List, *Evaluator) (e.Expr, error), env *environment) {
	bindFuncAtBottomAs(e.Identifier(name), Function{&f}, env)
}

func bindIdentifier(variable e.Expr, value e.Expr, env *environment) (e.Expr, error) {
	key, ok := keyFromExpr(variable)
	if !ok {
		return nil, fmt.Errorf("define: bad syntax, no valid identifier given in %s", variable.Repr())
	}

	scope := currentScope(env)
	(*scope)[key] = value

	return value, nil
}

func assign(variable e.Expr, value e.Expr, env *environment) (e.Expr, error) {
	key, ok := keyFromExpr(variable)
	if !ok {
		return nil, fmt.Errorf("set!: expected an identifier, got %s", e.TypeName(variable))
	}

	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		_, ok := (*scope)[key]
		if ok {
			(*scope)[key] = value
			return value, nil
		}
	}

	return nil, fmt.Errorf("set!: assignment disallowed for identifier %s", key.Name)
}

func lookupIdentifier(ident e.Expr, env *environment) (e.Expr, error) {
	key, ok := keyFromExpr(ident)
	if !ok {
		return nil, fmt.Errorf("undefined identifier: %s", ident.Repr())
	}

	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		res, ok := (*scope)[key]
		if ok {
			return res, nil
		}
	}

	// A macro-introduced reference to an existing binding (e.g. a built-in)
	// carries marks from expansion but the binding was created without marks.
	// Fall back to name-only lookup so these references resolve correctly.
	if key.MarksKey != "" {
		plainKey := scopeKey{Name: key.Name, MarksKey: ""}
		for i := len(*env) - 1; i >= 0; i-- {
			scope := (*env)[i]
			res, ok := (*scope)[plainKey]
			if ok {
				return res, nil
			}
		}
	}

	suggestion := formatSuggestion(suggestIdentifiers(key.Name, env))
	return nil, fmt.Errorf("undefined identifier: %s%s", key.Name, suggestion)
}

func newEnvWithEmptyScope(env *environment) *environment {
	new_env := append(*env, &scope{})
	return &new_env
}

func currentScope(env *environment) *scope {
	return (*env)[len(*env)-1]
}

func bottomScope(env *environment) *scope {
	return (*env)[0]
}
