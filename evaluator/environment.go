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

// RegisterExpr binds an arbitrary expression value (e.g. a SyntaxTransformer)
// to a name in the bottom scope of the environment.
func (env environment) RegisterExpr(name string, val e.Expr) {
	scope := bottomScope(&env)
	(*scope)[keyFromIdentifier(e.Identifier(name))] = val
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

func (env environment) LookupByName(name string) (e.Expr, error) {
	return lookupIdentifier(e.Identifier(name), &env)
}

// newModuleEnvironment creates a fresh environment that shares the builtins
// (bottom scope) from the parent but has its own top-level scope.
func newModuleEnvironment(parent *environment) *environment {
	builtins := bottomScope(parent)
	newEnv := environment{builtins, &scope{}}
	return &newEnv
}

// extractExports returns all bindings from the top scope (not builtins).
func extractExports(env *environment) *ModuleExports {
	topScope := currentScope(env)
	exports := &ModuleExports{
		Names:    make([]string, 0, len(*topScope)),
		Bindings: make(map[string]e.Expr, len(*topScope)),
	}
	for key, val := range *topScope {
		exports.Names = append(exports.Names, key.Name)
		exports.Bindings[key.Name] = val
	}
	return exports
}

func newEnvWithEmptyScope(env *environment) *environment {
	// Copy the slice to avoid aliasing the underlying array.
	// Without this, multiple calls to newEnvWithEmptyScope with the same
	// parent env can overwrite each other's scopes when the parent slice
	// has spare capacity.
	copied := make(environment, len(*env), len(*env)+1)
	copy(copied, *env)
	newEnv := append(copied, &scope{})
	return &newEnv
}

func currentScope(env *environment) *scope {
	return (*env)[len(*env)-1]
}

func bottomScope(env *environment) *scope {
	return (*env)[0]
}
