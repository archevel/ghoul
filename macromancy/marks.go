package macromancy

type Mark = uint64

type MarkSet map[Mark]bool

func NewMarkSet() MarkSet {
	return MarkSet{}
}

// Toggle returns a new MarkSet with the given mark flipped.
func (ms MarkSet) Toggle(m Mark) MarkSet {
	result := MarkSet{}
	for k, v := range ms {
		result[k] = v
	}
	if result[m] {
		delete(result, m)
	} else {
		result[m] = true
	}
	return result
}

func (ms MarkSet) IsEmpty() bool {
	return len(ms) == 0
}

func copyMarks(ms MarkSet) MarkSet {
	result := MarkSet{}
	for k, v := range ms {
		result[k] = v
	}
	return result
}
