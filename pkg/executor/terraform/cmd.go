package terraform

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

func terraform(ctx context.Context, env []string, args ...string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Env = append(os.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("could not get stdout pipe: %v", err)
	}
	var (
		errOut bytes.Buffer
	)
	cmd.Stderr = &errOut

	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, errOut.String())
	}

	var output []string

	s := bufio.NewScanner(stdout)
	for s.Scan() {
		line := s.Text()
		fmt.Println(line)
		output = append(output, line)
	}

	cmd.Wait()

	return output, s.Err()
}
