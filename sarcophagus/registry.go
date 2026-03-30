package sarcophagus

type Mummy struct {
	Names    []string
	Register func(prefix string, only map[string]bool, register func(string, interface{}))
}

var registry = map[string]*Mummy{}

func Entomb(shortName string, fullPath string, mummy *Mummy) {
	registry[shortName] = mummy
	if fullPath != shortName {
		registry[fullPath] = mummy
	}
}

func Unearth(name string) *Mummy {
	return registry[name]
}

func ClearRegistry() {
	for k := range registry {
		delete(registry, k)
	}
}

func RegisterIfAllowed(prefix string, only map[string]bool, name string, fn interface{}, register func(string, interface{})) {
	if only != nil && !only[name] {
		return
	}
	register(prefix+":"+name, fn)
}
