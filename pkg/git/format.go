package git

import (
	"errors"
	"fmt"
	"strings"
)

func firstField(lines []string, errText string) (string, error) {
	if len(lines) == 0 {
		return "", errors.New(errText)
	}

	fields := strings.Fields(lines[0])
	if len(fields) == 0 {
		return "", errors.New(errText)
	}

	if len(fields[0]) == 0 {
		return "", errors.New(errText)
	}

	return fields[0], nil
}

func formatRef(branch, tag string) string {
	if branch != "" {
		return formatRefForBranch(branch)
	}
	if tag != "" {
		return formatRefForTag(tag)
	}
	return ""
}

func formatRefForBranch(branch string) string {
	return fmt.Sprintf("refs/heads/%s", branch)
}

func formatRefForTag(tag string) string {
	return fmt.Sprintf("refs/tags/%s", tag)
}
