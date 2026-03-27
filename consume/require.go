package consume

import (
	"fmt"

	"github.com/archevel/ghoul/bones"
	"github.com/archevel/ghoul/mummy"
)

// requireContinuation handles (require module ...) with Node-based args.
func requireContinuation(args []*bones.Node) continuation {
	return func(arg *bones.Node, ev *Evaluator) (*bones.Node, error) {
		if len(args) == 0 {
			return bones.Nil, EvaluationError{msg: "require: module name required"}
		}

		moduleName := args[0].IdentName()
		if moduleName == "" {
			return bones.Nil, EvaluationError{msg: "require: module name must be an identifier"}
		}

		alias, only, err := parseRequireOptions(args)
		if err != nil {
			return bones.Nil, err
		}

		prefix := moduleName
		if alias != "" {
			prefix = alias
		}

		// Try Go sarcophagus first
		entry := mummy.LookupSarcophagus(moduleName)
		if entry != nil {
			requireKey := moduleName + ":" + prefix
			if ev.requiredModules[requireKey] {
				return bones.Nil, nil
			}
			result, err := requireSarcophagus(entry, prefix, only, ev)
			if err == nil {
				ev.requiredModules[requireKey] = true
			}
			return result, err
		}

		// Try Ghoul file module
		if ev.moduleState != nil {
			return requireGhoulModule(moduleName, prefix, only, ev)
		}

		return bones.Nil, EvaluationError{msg: fmt.Sprintf("require: module '%s' not found", moduleName)}
	}
}

func parseRequireOptions(args []*bones.Node) (alias string, only map[string]bool, err error) {
	i := 1 // skip module name
	for i < len(args) {
		keyword := args[i].IdentName()
		switch keyword {
		case "as":
			i++
			if i >= len(args) {
				return "", nil, EvaluationError{msg: "require: expected alias after 'as'"}
			}
			alias = args[i].IdentName()
			if alias == "" {
				return "", nil, EvaluationError{msg: "require: alias must be an identifier"}
			}
			i++
		case "only":
			i++
			if i >= len(args) {
				return "", nil, EvaluationError{msg: "require: expected name list after 'only'"}
			}
			nameList := args[i]
			if nameList.Kind != bones.ListNode {
				return "", nil, EvaluationError{msg: "require: 'only' must be followed by a list of names"}
			}
			only = map[string]bool{}
			for _, child := range nameList.Children {
				name := child.IdentName()
				if name == "" {
					return "", nil, EvaluationError{msg: "require: 'only' list must contain identifiers"}
				}
				only[name] = true
			}
			i++
		default:
			return "", nil, EvaluationError{msg: fmt.Sprintf("require: unexpected keyword '%s'", keyword)}
		}
	}
	return alias, only, nil
}

func checkNameConflict(qualifiedName string, ev *Evaluator) error {
	if _, err := lookupNode(bones.IdentNode(qualifiedName), ev.env); err == nil {
		return EvaluationError{msg: fmt.Sprintf("require: name '%s' already defined", qualifiedName)}
	}
	return nil
}

func requireSarcophagus(entry *mummy.SarcophagusEntry, prefix string, only map[string]bool, ev *Evaluator) (*bones.Node, error) {
	namesToRegister := entry.Names
	if only != nil {
		namesToRegister = make([]string, 0)
		for _, n := range entry.Names {
			if only[n] {
				namesToRegister = append(namesToRegister, n)
			}
		}
	}
	for _, name := range namesToRegister {
		if err := checkNameConflict(prefix+":"+name, ev); err != nil {
			return bones.Nil, err
		}
	}

	register := func(name string, fn interface{}) {
		if newFn, ok := fn.(func([]*bones.Node, *Evaluator) (*bones.Node, error)); ok {
			wrapped := func(args []*bones.Node, ev bones.Evaluator) (*bones.Node, error) {
				return newFn(args, ev.(*Evaluator))
			}
			bindNode(bones.IdentNode(name), bones.FuncNode(wrapped), ev.env)
		}
	}
	entry.Register(prefix, only, register)
	return bones.Nil, nil
}

func requireGhoulModule(moduleName string, prefix string, only map[string]bool, ev *Evaluator) (*bones.Node, error) {
	filePath, err := ev.moduleState.ResolveFile(moduleName)
	if err != nil {
		return bones.Nil, EvaluationError{msg: fmt.Sprintf("require: %s", err)}
	}

	if cached := ev.moduleState.GetCached(filePath); cached != nil {
		return registerModuleExports(cached, prefix, only, ev)
	}

	if err := ev.moduleState.CheckCycle(filePath); err != nil {
		return bones.Nil, EvaluationError{msg: fmt.Sprintf("require: %s", err)}
	}

	if ev.moduleLoader == nil {
		return bones.Nil, EvaluationError{msg: "require: no module loader configured"}
	}

	childState := ev.moduleState.ForChild(filePath)
	childState.BeginLoading(filePath)

	moduleEnv := newModuleEnvironment(ev.env)
	exports, err := ev.moduleLoader(filePath, moduleEnv, childState)
	childState.FinishLoading(filePath)
	if err != nil {
		return bones.Nil, EvaluationError{msg: fmt.Sprintf("require: error in module %s: %s", moduleName, err)}
	}

	ev.moduleState.CacheModule(filePath, exports)
	return registerModuleExports(exports, prefix, only, ev)
}

func registerModuleExports(exports *ModuleExports, prefix string, only map[string]bool, ev *Evaluator) (*bones.Node, error) {
	for _, name := range exports.Names {
		if only != nil && !only[name] {
			continue
		}
		qualifiedName := prefix + ":" + name
		if err := checkNameConflict(qualifiedName, ev); err != nil {
			return bones.Nil, err
		}
		bindNode(bones.IdentNode(qualifiedName), exports.Bindings[name], ev.env)
	}
	ev.requiredModules[prefix+":ghoul"] = true
	return bones.Nil, nil
}
