//go:build e2e

package shell

import (
	"fmt"
	"strings"
)

type Command []string

func DiskUsageWithTotal(directory string) Command {
	return Command{"du", "-c", directory}
}

func FilterLastLineOnly() Command {
	return Command{"tail", "-n", "1"}
}

func Pipe(source []string, sink []string) Command {
	piped := source
	piped = append(piped, "|")
	piped = append(piped, sink...)
	return piped
}

func ReadFile(path string) Command {
	return Command{"cat", path}
}

func ListDirectory(path string) Command {
	return Command{"ls", path}
}

func CheckIfEmpty(path string) Command {
	command := fmt.Sprintf("[ \"$(ls -A %s)\" ] || return 1", path)
	return Command{command}
}

func Shell(command Command) Command {
	return Command{"sh", "-c", command.String()}
}

func (c Command) String() string {
	return strings.Join(c, " ")
}

func Echo(msg string) Command {
	return Command{"echo", msg}
}
