package kubeobjects

var exists = struct{}{}

func NewStringSet() *StringSet {
	set := StringSet{}
	set.elements = map[string]struct{}{}
	return &set
}

// TODO: Make it generic when linter is fixed
type StringSet struct {
	elements map[string]struct{}
}

func (set StringSet) Add(element string) {
	set.elements[element] = exists
}

func (set StringSet) Remove(element string) {
	delete(set.elements, element)
}

func (set StringSet) Contains(element string) bool {
	_, ok := set.elements[element]
	return ok
}

func (set StringSet) Size() int {
	return len(set.elements)
}
