package bash

import (
	"fmt"
	"strings"
)

type BashCmd []string

func DiskUsageWithTotal(directory string) BashCmd {
	return BashCmd{"du", "-c", directory}
}

func FilterLastLineOnly() []string {
	return BashCmd{"tail", "-n", "1"}
}

func Pipe(source []string, sink []string) BashCmd {
	piped := source
	piped = append(piped, "|")
	piped = append(piped, sink...)
	return piped
}

func ReadFile(path string) BashCmd {
	return BashCmd{"cat", path}
}

func ListDirectory(path string) BashCmd {
	return BashCmd{"ls", path}
}

func Shell(command BashCmd) BashCmd {
	return BashCmd{"sh", "-c", command.String()}
}

func (c BashCmd) String() string {
	cmdString := ""
	for _, str := range c {
		cmdString = fmt.Sprintf("%s %s", cmdString, str)
	}
	return strings.Trim(cmdString, " ")
}
