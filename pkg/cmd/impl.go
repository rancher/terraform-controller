package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

//Execute process command without logging text to standard output
func Execute(command Command) (string, error) {
	var allOutput []string
	err := execute(command, &allOutput, &allOutput)

	output := strings.Join(allOutput, "\n")
	return output, err
}

func execute(command Command, storedStdout *[]string, storedStderr *[]string) error {
	cmd := exec.Command(command.Command, command.Args...)
	cmd.Dir = command.WorkingDir
	cmd.Stdin = os.Stdin
	cmd.Env = formatEnvVars(command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	if err := readStdoutAndStderr(stdout, stderr, storedStdout, storedStderr, 1024); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func formatEnvVars(command Command) []string {
	env := os.Environ()
	for key, value := range command.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func readStdoutAndStderr(stdout io.ReadCloser, stderr io.ReadCloser, storedStdout *[]string, storedStderr *[]string, maxLineSize int) error {
	stdoutScanner := bufio.NewScanner(stdout)
	stderrScanner := bufio.NewScanner(stderr)

	if maxLineSize > 0 {
		stdoutScanner.Buffer(make([]byte, maxLineSize), maxLineSize)
		stderrScanner.Buffer(make([]byte, maxLineSize), maxLineSize)
	}

	wg := &sync.WaitGroup{}
	mutex := &sync.Mutex{}
	wg.Add(2)
	go readData(stdoutScanner, wg, mutex, storedStdout)
	go readData(stderrScanner, wg, mutex, storedStderr)
	wg.Wait()

	if err := stdoutScanner.Err(); err != nil {
		return err
	}

	if err := stderrScanner.Err(); err != nil {
		return err
	}

	return nil
}

func readData(scanner *bufio.Scanner, wg *sync.WaitGroup, mutex *sync.Mutex, allOutput *[]string) {
	defer wg.Done()
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		logTextAndAppendToOutput(mutex, scanner.Text(), allOutput)
	}
}

func logTextAndAppendToOutput(mutex *sync.Mutex, text string, allOutput *[]string) {
	defer mutex.Unlock()
	mutex.Lock()
	*allOutput = append(*allOutput, text)
}
