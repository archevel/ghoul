package consume

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	e "github.com/archevel/ghoul/bones"
)

// scopeKey must be comparable for use as a map key, so marks are
// encoded as a canonical sorted string rather than a map.
type scopeKey struct {
	Name     string
	MarksKey string
}

func keyFromName(name string) scopeKey {
	return scopeKey{Name: name, MarksKey: ""}
}

func keyFromNameAndMarks(name string, marks map[uint64]bool) scopeKey {
	return scopeKey{Name: name, MarksKey: canonicalMarks(marks)}
}

func keyFromNode(node *e.Node) (scopeKey, bool) {
	if node.Kind == e.IdentifierNode {
		if len(node.Marks) > 0 {
			return keyFromNameAndMarks(node.Name, node.Marks), true
		}
		return keyFromName(node.Name), true
	}
	return scopeKey{}, false
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

type scope map[scopeKey]*e.Node

type Environment = environment
type environment []*scope

func NewEnvironment() *environment {
	return newEnvWithEmptyScope(&environment{})
}

// BoundIdentifierNames returns a map of all identifier names currently bound
// in the environment, plus special form keywords.
func (env environment) BoundIdentifierNames() map[string]bool {
	result := map[string]bool{
		"cond": true, "else": true, "begin": true, "lambda": true,
		"define": true, "set!": true, "define-syntax": true,
		"syntax-rules": true, "quote": true,
	}
	for i := range env {
		for key := range *env[i] {
			if key.MarksKey == "" {
				result[key.Name] = true
			}
		}
	}
	return result
}

func (env environment) Register(name string, f func([]*e.Node, *Evaluator) (*e.Node, error)) {
	scope := bottomScope(&env)
	wrapped := func(args []*e.Node, ev e.Evaluator) (*e.Node, error) {
		return f(args, ev.(*Evaluator))
	}
	(*scope)[keyFromName(name)] = e.FuncNode(wrapped)
}

func RegisterFuncAs(name string, f func([]*e.Node, *Evaluator) (*e.Node, error), env *environment) {
	scope := bottomScope(env)
	wrapped := func(args []*e.Node, ev e.Evaluator) (*e.Node, error) {
		return f(args, ev.(*Evaluator))
	}
	(*scope)[keyFromName(name)] = e.FuncNode(wrapped)
}

func bindNode(variable *e.Node, value *e.Node, env *environment) (*e.Node, error) {
	key, ok := keyFromNode(variable)
	if !ok {
		return nil, fmt.Errorf("define: bad syntax, no valid identifier given in %s", variable.Repr())
	}
	scope := currentScope(env)
	(*scope)[key] = value
	return value, nil
}

func assignByName(variable *e.Node, value *e.Node, env *environment) (*e.Node, error) {
	key, ok := keyFromNode(variable)
	if !ok {
		return nil, fmt.Errorf("set!: expected an identifier, got %s", e.NodeTypeName(variable))
	}
	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		if _, ok := (*scope)[key]; ok {
			(*scope)[key] = value
			return value, nil
		}
	}
	return nil, fmt.Errorf("set!: assignment disallowed for identifier %s", key.Name)
}

func lookupNode(ident *e.Node, env *environment) (*e.Node, error) {
	key, ok := keyFromNode(ident)
	if !ok {
		return nil, fmt.Errorf("undefined identifier: %s", ident.Repr())
	}

	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		if res, ok := (*scope)[key]; ok {
			return res, nil
		}
	}

	// Fall back to name-only lookup for macro-introduced references
	if key.MarksKey != "" {
		plainKey := scopeKey{Name: key.Name, MarksKey: ""}
		for i := len(*env) - 1; i >= 0; i-- {
			scope := (*env)[i]
			if res, ok := (*scope)[plainKey]; ok {
				return res, nil
			}
		}
	}

	suggestion := formatSuggestion(suggestIdentifiers(key.Name, env))
	return nil, fmt.Errorf("undefined identifier: %s%s", key.Name, suggestion)
}

func (env environment) LookupByName(name string) (*e.Node, error) {
	return lookupNode(e.IdentNode(name), &env)
}

// BindByName binds a value to a name in the current (top) scope.
func (env *environment) BindByName(name string, val *e.Node) {
	scope := currentScope(env)
	(*scope)[keyFromName(name)] = val
}

// NewModuleEnvironment creates a fresh environment that shares the builtins
// (bottom scope) from the parent but has its own top-level scope.
func NewModuleEnvironment(parent *environment) *environment {
	builtins := bottomScope(parent)
	newEnv := environment{builtins, &scope{}}
	return &newEnv
}

// ExtractExports returns all bindings from the top scope (not builtins).
func ExtractExports(env *environment) *ModuleExports {
	topScope := currentScope(env)
	exports := &ModuleExports{
		Names:    make([]string, 0, len(*topScope)),
		Bindings: make(map[string]*e.Node, len(*topScope)),
	}
	for key, val := range *topScope {
		exports.Names = append(exports.Names, key.Name)
		exports.Bindings[key.Name] = val
	}
	return exports
}

func newEnvWithEmptyScope(env *environment) *environment {
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
