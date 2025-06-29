package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// cmdName1 and cmdName2 are the names of the commands to be executed
// argv1 and argv2 are the arguments for the commands
// cmd1 and cmd2 are the actual command objects created using exec.Command
func HandlePipe(input string) {
	parts := strings.SplitN(input, "|", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	executePipeline(strings.Split(left, " "), strings.Split(right, " "))
}

func executePipeline(cmd1Args []string, cmd2Args []string) error {
	filePath1, exists1 := findBinInPath(cmd1Args[0])
	if !exists1 {
		return fmt.Errorf("%s: command not found", cmd1Args[0])
	}
	filePath2, exists2 := findBinInPath(cmd2Args[0])
	if !exists2 {
		return fmt.Errorf("%s: command not found", cmd2Args[0])
	}

	cmd1 := exec.Command(filePath1, cmd1Args[1:]...)
	cmd2 := exec.Command(filePath2, cmd2Args[1:]...)

	r, w := io.Pipe()
	cmd1.Stdout = w
	cmd2.Stdin = r
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr

	if err := cmd1.Start(); err != nil {
		return err
	}
	if err := cmd2.Start(); err != nil {
		return err
	}

	go func() {
		cmd1.Wait()
		w.Close()
	}()

	return cmd2.Wait()
}
