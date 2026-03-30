package reanimator

import (
	"fmt"

	"github.com/archevel/ghoul/bones"
	ev "github.com/archevel/ghoul/consume"
	"github.com/archevel/ghoul/sarcophagus"
)

// processRequire handles (require module ...) during macro expansion.
// It loads sarcophagi and Ghoul modules eagerly, binding runtime exports
// into the reanimator's eval environment and macro exports into the
// current macro scope. Returns nil to strip the require from output.
func (exp *Reanimator) processRequire(node *bones.Node, scope *macroScope) (*bones.Node, error) {
	args := node.Children[1:]
	if len(args) == 0 {
		return nil, fmt.Errorf("require: module name required")
	}

	moduleName := args[0].IdentName()
	if moduleName == "" {
		return nil, fmt.Errorf("require: module name must be an identifier")
	}

	alias, only, err := parseRequireOptions(args)
	if err != nil {
		return nil, err
	}

	prefix := moduleName
	if alias != "" {
		prefix = alias
	}

	// Try entombed mummy first
	mummy := sarcophagus.Unearth(moduleName)
	if mummy != nil {
		requireKey := moduleName + ":" + prefix
		if exp.requiredModules[requireKey] {
			return nil, nil
		}
		err := exp.requireMummy(mummy, prefix, only)
		if err == nil {
			exp.requiredModules[requireKey] = true
		}
		return nil, err
	}

	// Try Ghoul file module
	if exp.moduleState != nil {
		return nil, exp.requireGhoulModule(moduleName, prefix, only, scope)
	}

	return nil, fmt.Errorf("require: module '%s' not found", moduleName)
}

func parseRequireOptions(args []*bones.Node) (alias string, only map[string]bool, err error) {
	i := 1 // skip module name
	for i < len(args) {
		keyword := args[i].IdentName()
		switch keyword {
		case "as":
			i++
			if i >= len(args) {
				return "", nil, fmt.Errorf("require: expected alias after 'as'")
			}
			alias = args[i].IdentName()
			if alias == "" {
				return "", nil, fmt.Errorf("require: alias must be an identifier")
			}
			i++
		case "only":
			i++
			if i >= len(args) {
				return "", nil, fmt.Errorf("require: expected name list after 'only'")
			}
			nameList := args[i]
			if nameList.Kind != bones.ListNode {
				return "", nil, fmt.Errorf("require: 'only' must be followed by a list of names")
			}
			only = map[string]bool{}
			for _, child := range nameList.Children {
				name := child.IdentName()
				if name == "" {
					return "", nil, fmt.Errorf("require: 'only' list must contain identifiers")
				}
				only[name] = true
			}
			i++
		default:
			return "", nil, fmt.Errorf("require: unexpected keyword '%s'", keyword)
		}
	}
	return alias, only, nil
}

func (exp *Reanimator) checkNameConflict(qualifiedName string) error {
	if _, err := exp.evalEnv.LookupByName(qualifiedName); err == nil {
		return fmt.Errorf("require: name '%s' already defined", qualifiedName)
	}
	return nil
}

func (exp *Reanimator) requireMummy(mummy *sarcophagus.Mummy, prefix string, only map[string]bool) error {
	namesToRegister := mummy.Names
	if only != nil {
		namesToRegister = make([]string, 0)
		for _, n := range mummy.Names {
			if only[n] {
				namesToRegister = append(namesToRegister, n)
			}
		}
	}
	for _, name := range namesToRegister {
		if err := exp.checkNameConflict(prefix + ":" + name); err != nil {
			return err
		}
	}

	register := func(name string, fn interface{}) {
		if newFn, ok := fn.(func([]*bones.Node, *ev.Evaluator) (*bones.Node, error)); ok {
			wrapped := func(args []*bones.Node, evaluator bones.Evaluator) (*bones.Node, error) {
				return newFn(args, evaluator.(*ev.Evaluator))
			}
			exp.evalEnv.BindByName(name, bones.FuncNode(wrapped))
		}
	}
	mummy.Register(prefix, only, register)
	return nil
}

func (exp *Reanimator) requireGhoulModule(moduleName string, prefix string, only map[string]bool, scope *macroScope) error {
	filePath, err := exp.moduleState.ResolveFile(moduleName)
	if err != nil {
		return fmt.Errorf("require: %s", err)
	}

	if cached := exp.moduleState.GetCached(filePath); cached != nil {
		return exp.registerModuleExports(cached, prefix, only, scope)
	}

	if err := exp.moduleState.CheckCycle(filePath); err != nil {
		return fmt.Errorf("require: %s", err)
	}

	if exp.moduleLoader == nil {
		return fmt.Errorf("require: no module loader configured")
	}

	childState := exp.moduleState.ForChild(filePath)
	childState.BeginLoading(filePath)

	exports, err := exp.moduleLoader(filePath, exp)
	childState.FinishLoading(filePath)
	if err != nil {
		return fmt.Errorf("require: error in module %s: %s", moduleName, err)
	}

	exp.moduleState.CacheModule(filePath, exports)
	return exp.registerModuleExports(exports, prefix, only, scope)
}

func (exp *Reanimator) registerModuleExports(exports *ev.ModuleExports, prefix string, only map[string]bool, scope *macroScope) error {
	// Register runtime exports — skip conflict check for values that
	// are already bound (happens when the same cached module is required
	// from multiple modules with the same prefix)
	for _, name := range exports.Names {
		if only != nil && !only[name] {
			continue
		}
		qualifiedName := prefix + ":" + name
		if existing, err := exp.evalEnv.LookupByName(qualifiedName); err == nil {
			if existing == exports.Bindings[name] {
				continue // same binding, harmless re-require
			}
			return fmt.Errorf("require: name '%s' already defined", qualifiedName)
		}
		exp.evalEnv.BindByName(qualifiedName, exports.Bindings[name])
	}

	// Register macro exports into the current macro scope
	for name, binding := range exports.Macros {
		if only != nil && !only[name] {
			continue
		}
		qualifiedName := prefix + ":" + name
		if mb, ok := binding.(macroBinding); ok {
			scope.define(qualifiedName, mb)
		}
	}

	exp.requiredModules[prefix+":ghoul"] = true
	return nil
}
