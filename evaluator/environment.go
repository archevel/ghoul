package evaluator

import (
	"errors"
	e "github.com/archevel/ghoul/expressions"
)

type frame map[e.Identifier]e.Expr

type environment []*frame

func NewEnvironment() *environment {
	return newEnvWithEmptyFrame(&environment{})
}

func bindFuncAtBottomAs(id e.Identifier, fun e.Function, env *environment) {
	frame := bottomFrame(env)
	(*frame)[id] = fun
}

func RegisterFuncAs(name string, f func(e.List) (e.Expr, error), env *environment) {
	bindFuncAtBottomAs(e.Identifier(name), e.Function{&f}, env)
}

func bindIdentifier(variable e.Expr, value e.Expr, env *environment) (e.Expr, error) {

	id, id_ok := variable.(e.Identifier)
	if !id_ok {
		return nil, errors.New("Bad syntax: no valid identifier given")
	}

	frame := currentFrame(env)
	(*frame)[id] = value

	//	fmt.Println("bound id:", id.Repr(), "to:", value.Repr())
	return value, nil
}

func assign(variable e.Expr, value e.Expr, env *environment) (e.Expr, error) {

	ident := variable.(e.Identifier)

	for i := len(*env) - 1; i >= 0; i-- {
		frame := (*env)[i]
		_, ok := (*frame)[ident]
		if ok {
			(*frame)[ident] = value
			return value, nil
		}
	}

	return nil, errors.New("assignment disallowed")
}

func lookupIdentifier(ident e.Identifier, env *environment) (e.Expr, error) {
	for i := len(*env) - 1; i >= 0; i-- {
		frame := (*env)[i]
		res, ok := (*frame)[ident]
		if ok {
			return res, nil
		}
	}

	return nil, errors.New("undefined identifier: " + string(ident))

}

func newEnvWithEmptyFrame(env *environment) *environment {
	new_env := append(*env, &frame{})
	return &new_env
}

func currentFrame(env *environment) *frame {
	return (*env)[len(*env)-1]
}

func bottomFrame(env *environment) *frame {
	return (*env)[0]
}
