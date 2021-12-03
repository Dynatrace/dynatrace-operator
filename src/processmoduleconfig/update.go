package processmoduleconfig

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/afero"
)

// ConfMap is the representation of a config file with sections that are divided by headers (map[header] == section)
// each section consists of key value pairs.
type ConfMap map[string]map[string]string

// example match: [general]
var sectionRegexp, _ = regexp.Compile(`\[(.*)\]`)

// Update opens the file at `sourcePath` and merges it with the ConfMap provided
// then writes the results into a file at `destPath`
func Update(fs afero.Fs, sourcePath, destPath string, conf ConfMap) error {
	fileInfo, err := fs.Stat(sourcePath)
	if err != nil {
		return err
	}
	sourceFile, err := fs.Open(sourcePath)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(sourceFile)
	currentSection := ""
	content := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if header := confSectionHeader(line); header != "" {
			leftovers := addLeftoversForSection(currentSection, conf)
			if hasExtraNewLine(leftovers, content) {
				content = content[:len(content)-1]
				leftovers = append(leftovers, "")
			}
			content = append(content, leftovers...)
			currentSection = header
			content = append(content, line)
		} else if strings.HasPrefix(line, "#") {
			content = append(content, line)
		} else {
			content = append(content, mergeLine(line, currentSection, conf))
		}
	}
	if err := scanner.Err(); err != nil {
		sourceFile.Close()
		return err
	}

	// the last section's leftover cleanup never runs in the for loop
	leftovers := addLeftoversForSection(currentSection, conf)
	if hasExtraNewLine(leftovers, content) {
		content = content[:len(content)-1]
	}
	content = append(content, leftovers...)

	// for sections not in the original conf file need to be added as well
	leftovers = addLeftovers(conf)
	if hasMissingNewLine(leftovers, content) {
		content = append(content, "")
	}
	content = append(content, leftovers...)

	if err = sourceFile.Close(); err != nil {
		return err
	}

	return storeFile(fs, destPath, fileInfo.Mode(), content)
}

func hasExtraNewLine(leftovers, content []string) bool {
	return len(leftovers) != 0 && len(content) != 0 && content[len(content)-1] == ""
}

func hasMissingNewLine(leftovers, content []string) bool {
	return len(leftovers) != 0 && len(content) != 0 && content[len(content)-1] != ""
}

func storeFile(fs afero.Fs, destPath string, fileMode fs.FileMode, content []string) error {
	file, err := fs.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}
	for _, line := range content {
		_, _ = file.WriteString(line + "\n")
		if err != nil {
			file.Close()
			return err
		}
	}
	return file.Close()
}

func confSectionHeader(confLine string) string {
	if matches := sectionRegexp.FindStringSubmatch(confLine); len(matches) != 0 {
		return matches[1]
	}
	return ""
}

func addLeftovers(conf ConfMap) []string {
	lines := []string{}
	for section, props := range conf {
		lines = append(lines, fmt.Sprintf("[%s]", section))
		for key, value := range props {
			lines = append(lines, fmt.Sprintf("%s %s", key, value))
		}
	}
	return lines
}

func addLeftoversForSection(currentSection string, conf ConfMap) []string {
	lines := []string{}
	if currentSection != "" {
		section, ok := conf[currentSection]
		if ok {
			for key, value := range section {
				lines = append(lines, fmt.Sprintf("%s %s", key, value))
			}
			delete(conf, currentSection)
		}
	}
	return lines
}

func mergeLine(line, currentSection string, conf ConfMap) string {
	splitLine := strings.Split(line, " ")
	key := splitLine[0]

	props, ok := conf[currentSection]
	if !ok {
		return line
	}
	newValue, ok := props[key]
	if !ok {
		return line
	}
	delete(conf[currentSection], key)
	return fmt.Sprintf("%s %s", key, newValue)
}
