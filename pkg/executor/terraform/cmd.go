package terraform

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func terraform(ctx context.Context, env []string, args ...string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Env = append(os.Environ(), env...)

	var (
		out    bytes.Buffer
		errOut bytes.Buffer
	)
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrap(err, errOut.String())
	}

	var output []string
	s := bufio.NewScanner(&out)
	for s.Scan() {
		line := s.Text()
		fmt.Println(line)
		output = append(output, line)
	}

	return output, s.Err()
}
