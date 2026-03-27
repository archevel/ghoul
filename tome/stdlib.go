package tome

import (
	ev "github.com/archevel/ghoul/consume"
)

func RegisterAll(env *ev.Environment) {
	registerArithmetic(env)
	registerComparison(env)
	registerLogic(env)
	registerStrings(env)
	registerLists(env)
	registerTypes(env)
	registerIO(env)
	registerSyntax(env)
	registerConversions(env)
}
