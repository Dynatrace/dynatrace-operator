package arg

import "fmt"

type Arg struct {
	Name  string
	Value string
}

func (a Arg) String() string {
	return fmt.Sprintf("--%s=%s", a.Name, a.Value)
}

func ConvertArgsToStrings(args []Arg) []string {
	stringargs := make([]string, len(args))
	for i, arg := range args {
		stringargs[i] = arg.String()
	}

	return stringargs
}
