package evaluator

import (
	"errors"

	e "github.com/archevel/ghoul/expressions"
)

type scope map[e.Identifier]e.Expr

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
	(*scope)[id] = fun
}

func RegisterFuncAs(name string, f func(e.List, *Evaluator) (e.Expr, error), env *environment) {
	bindFuncAtBottomAs(e.Identifier(name), Function{&f}, env)
}

func bindIdentifier(variable e.Expr, value e.Expr, env *environment) (e.Expr, error) {

	id, id_ok := variable.(e.Identifier)
	if !id_ok {
		return nil, errors.New("Bad syntax: no valid identifier given")
	}

	scope := currentScope(env)
	(*scope)[id] = value

	return value, nil
}

func assign(variable e.Expr, value e.Expr, env *environment) (e.Expr, error) {

	ident := variable.(e.Identifier)

	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		_, ok := (*scope)[ident]
		if ok {
			(*scope)[ident] = value
			return value, nil
		}
	}

	return nil, errors.New("assignment disallowed")
}

func lookupIdentifier(ident e.Identifier, env *environment) (e.Expr, error) {
	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		res, ok := (*scope)[ident]
		if ok {
			return res, nil
		}
	}

	return nil, errors.New("undefined identifier: " + string(ident))

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
