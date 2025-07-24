package arg

import "fmt"

type Arg struct {
	Name  string
	Value string
}

func (a Arg) String() string {
	if a.Value == "" {
		return "--" + a.Name
	}

	return fmt.Sprintf("--%s=%s", a.Name, a.Value)
}

func ConvertArgsToStrings(args []Arg) []string {
	convertedArgs := make([]string, len(args))
	for i, arg := range args {
		convertedArgs[i] = arg.String()
	}

	return convertedArgs
}
