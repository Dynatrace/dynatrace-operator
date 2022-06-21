package main

import (
	"io/ioutil"
	"regexp"
	"strings"
)

func sed(path, pattern, replacement string) error {
	patternRegex, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	fileString := string(fileBytes)
	fileLines := strings.Split(fileString, "\n")

	for i := range fileLines {
		if patternRegex.MatchString(fileLines[i]) {
			fileLines[i] = patternRegex.ReplaceAllString(fileLines[i], replacement)
		}
	}
	fileString = strings.Join(fileLines, "\n")
	fileBytes = []byte(fileString)
	return ioutil.WriteFile(path+".output", fileBytes, 0o644)
}

func main() {
	sed("file.txt", "^version: .*", "version: 9")
}
