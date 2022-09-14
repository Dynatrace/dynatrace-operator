package bash

import "fmt"

func DiskUsageWithTotal(directory string) string {
	return fmt.Sprintf("du -c %s", directory)
}

func FilterLastLineOnly() string {
	return "tail -n 1"
}

func Pipe(source string, sink string) string {
	return fmt.Sprintf("%s | %s", source, sink)
}

func ReadFile(path string) string {
	return fmt.Sprintf("cat %s", path)
}

func ListDirectory(path string) string {
	return fmt.Sprintf("ls %s", path)
}
