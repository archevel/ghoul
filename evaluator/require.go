package evaluator

import (
	"fmt"
	"os"

	e "github.com/archevel/ghoul/expressions"
	"github.com/archevel/ghoul/mummy"
	"github.com/archevel/ghoul/parser"
)

// requireContinuationFor handles (require module), (require module as alias),
// (require module only (name ...)), and (require module as alias only (name ...)).
func requireContinuationFor(args e.List) continuation {
	return func(arg e.Expr, ev *Evaluator) (e.Expr, error) {
		if args == e.NIL {
			return e.NIL, NewEvaluationError("require: module name required", args)
		}

		moduleName, ok := args.First().(e.Identifier)
		if !ok {
			return e.NIL, NewEvaluationError("require: module name must be an identifier", args)
		}

		alias, only, err := parseRequireOptions(args)
		if err != nil {
			return e.NIL, err
		}

		prefix := string(moduleName)
		if alias != "" {
			prefix = alias
		}

		// Try Go sarcophagus first
		entry := mummy.LookupSarcophagus(string(moduleName))
		if entry != nil {
			requireKey := string(moduleName) + ":" + prefix
			if ev.requiredModules[requireKey] {
				return e.NIL, nil
			}
			result, err := requireSarcophagus(entry, prefix, only, ev, args)
			if err == nil {
				ev.requiredModules[requireKey] = true
			}
			return result, err
		}

		// Try Ghoul file module
		if ev.moduleState != nil {
			return requireGhoulModule(string(moduleName), prefix, only, ev, args)
		}

		return e.NIL, NewEvaluationError(fmt.Sprintf("require: module '%s' not found", moduleName), args)
	}
}

func parseRequireOptions(args e.List) (alias string, only map[string]bool, err error) {
	rest := args.Second()

	for rest != e.NIL {
		restList, ok := rest.(e.List)
		if !ok {
			break
		}
		keyword, ok := restList.First().(e.Identifier)
		if !ok {
			break
		}

		switch keyword {
		case e.Identifier("as"):
			tail, ok := restList.Tail()
			if !ok || tail == e.NIL {
				return "", nil, NewEvaluationError("require: expected alias after 'as'", args)
			}
			aliasId, ok := tail.First().(e.Identifier)
			if !ok {
				return "", nil, NewEvaluationError("require: alias must be an identifier", args)
			}
			alias = string(aliasId)
			rest = tail.Second()
		case e.Identifier("only"):
			tail, ok := restList.Tail()
			if !ok || tail == e.NIL {
				return "", nil, NewEvaluationError("require: expected name list after 'only'", args)
			}
			nameList, ok := tail.First().(e.List)
			if !ok {
				return "", nil, NewEvaluationError("require: 'only' must be followed by a list of names", args)
			}
			only = map[string]bool{}
			for nameList != e.NIL {
				nameId, ok := nameList.First().(e.Identifier)
				if !ok {
					return "", nil, NewEvaluationError("require: 'only' list must contain identifiers", args)
				}
				only[string(nameId)] = true
				nameList, _ = nameList.Tail()
			}
			rest = tail.Second()
		default:
			return "", nil, NewEvaluationError(fmt.Sprintf("require: unexpected keyword '%s'", keyword), args)
		}
	}

	return alias, only, nil
}

func checkNameConflict(qualifiedName string, ev *Evaluator, args e.List) error {
	if _, err := lookupIdentifier(e.Identifier(qualifiedName), ev.env); err == nil {
		return NewEvaluationError(
			fmt.Sprintf("require: name '%s' already defined", qualifiedName), args)
	}
	return nil
}

func requireSarcophagus(entry *mummy.SarcophagusEntry, prefix string, only map[string]bool, ev *Evaluator, args e.List) (e.Expr, error) {
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
		if err := checkNameConflict(prefix+":"+name, ev, args); err != nil {
			return e.NIL, err
		}
	}

	register := func(name string, fn interface{}) {
		if ghoulFn, ok := fn.(func(e.List, *Evaluator) (e.Expr, error)); ok {
			bindIdentifier(e.Identifier(name), Function{&ghoulFn}, ev.env)
		}
	}
	entry.Register(prefix, only, register)
	return e.NIL, nil
}

func requireGhoulModule(moduleName string, prefix string, only map[string]bool, ev *Evaluator, args e.List) (e.Expr, error) {
	filePath, err := ev.moduleState.ResolveFile(moduleName)
	if err != nil {
		return e.NIL, WrapError(fmt.Sprintf("require: %s", err), args, err)
	}

	// Check cache
	if cached := ev.moduleState.GetCached(filePath); cached != nil {
		return registerModuleExports(cached, prefix, only, ev, args)
	}

	// Check cycles
	if err := ev.moduleState.CheckCycle(filePath); err != nil {
		return e.NIL, WrapError(fmt.Sprintf("require: %s", err), args, err)
	}

	// Load and parse the file.
	// The Open error path is not covered by tests because ResolveFile already
	// verifies the file exists via os.Stat, so Open only fails in unusual
	// conditions (permission changes, filesystem errors) that are hard to
	// reproduce reliably in tests.
	f, err := os.Open(filePath)
	if err != nil {
		return e.NIL, WrapError(fmt.Sprintf("require: failed to open %s: %s", filePath, err), args, err)
	}
	defer f.Close()

	parseRes, parsed := parser.ParseWithFilename(f, &filePath)
	if parseRes != 0 {
		return e.NIL, NewEvaluationError(fmt.Sprintf("require: failed to parse %s", filePath), args)
	}

	// Evaluate in a fresh module environment sharing builtins
	moduleEnv := newModuleEnvironment(ev.env)
	childState := ev.moduleState.ForChild(filePath)
	childState.BeginLoading(filePath)

	moduleEval := &Evaluator{
		log:             ev.log,
		env:             moduleEnv,
		requiredModules: map[string]bool{},
		moduleState:     childState,
		markCounter:     ev.markCounter,
	}

	_, err = moduleEval.Evaluate(parsed.Expressions)
	childState.FinishLoading(filePath)
	if err != nil {
		return e.NIL, WrapError(fmt.Sprintf("require: error in module %s: %s", moduleName, err), args, err)
	}

	// Extract top-level bindings as exports
	exports := extractExports(moduleEnv)
	ev.moduleState.CacheModule(filePath, exports)

	return registerModuleExports(exports, prefix, only, ev, args)
}

func registerModuleExports(exports *ModuleExports, prefix string, only map[string]bool, ev *Evaluator, args e.List) (e.Expr, error) {
	for _, name := range exports.Names {
		if only != nil && !only[name] {
			continue
		}
		qualifiedName := prefix + ":" + name
		if err := checkNameConflict(qualifiedName, ev, args); err != nil {
			return e.NIL, err
		}
		bindIdentifier(e.Identifier(qualifiedName), exports.Bindings[name], ev.env)
	}
	ev.requiredModules[prefix+":ghoul"] = true
	return e.NIL, nil
}
