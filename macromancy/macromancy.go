package macromancy

import (
	e "github.com/archevel/ghoul/expressions"
)

type Transformer interface {
	Transform(list e.List) (e.List, error)
}
type Macromancer struct{}

func (m Macromancer) Transform(list e.List) (e.List, error) {
	return list, nil
}
