// Package terraform to run terraform commands
package terraform

import (
	"context"
	"os"
	"strings"
)

const newLine = "\n"

func Apply() (string, error) {
	output, err := terraform(context.Background(), os.Environ(), "apply", "-input=false", "-auto-approve", "tfplan")
	if err != nil {
		return "", err
	}

	return combineOutput(output), nil
}

func Destroy() (string, error) {
	output, err := terraform(context.Background(), os.Environ(), "destroy", "-input=false", "-auto-approve")
	if err != nil {
		return "", err
	}

	return combineOutput(output), nil
}

func Init() (string, error) {
	output, err := terraform(context.Background(), os.Environ(), "init", "-input=false")
	if err != nil {
		return "", err
	}

	return combineOutput(output), nil
}

// Output runs 'terraform output -json' and returns the blob as a string
func Output() (string, error) {
	output, err := terraform(context.Background(), os.Environ(), "output", "-json")
	if err != nil {
		return "", err
	}

	return combineOutput(output), nil
}

// Plan runs 'terraform plan' with the destroy flag controlling the play type
func Plan(destroy bool) (string, error) {
	args := []string{"plan", "-input=false", "-out=tfplan"}
	if destroy {
		args = append(args, "-destroy")
	}

	output, err := terraform(context.Background(), os.Environ(), args...)
	if err != nil {
		return "", err
	}

	return combineOutput(output), nil
}

func combineOutput(in []string) string {
	var b strings.Builder
	for _, v := range in {
		b.WriteString(v)
		b.WriteString(newLine)
	}
	return b.String()
}
