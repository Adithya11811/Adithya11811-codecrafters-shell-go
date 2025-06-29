package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func isBuiltin(cmd string) bool {
	for _, b := range builtIns {
		if b == cmd {
			return true
		}
	}
	return false
}


// cmdName1 and cmdName2 are the names of the commands to be executed
// argv1 and argv2 are the arguments for the commands
// cmd1 and cmd2 are the actual command objects created using exec.Command
func HandlePipe(input string) {
	parts := strings.SplitN(input, "|", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	// cmd1Name, cmd1Args := splitWithQuoting(left)
	// cmd2Name, cmd2Args := splitWithQuoting(right)

	// Move(cmd1Name)
	// Move(cmd2Name)

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

// Pipeline handler that supports builtins on the left
func executePipelineBuiltinAware(cmd1Args []string, cmd2Args []string) error {
	r, w := io.Pipe()

	if isBuiltin(cmd1Args[0]) {
		go func() {
			switch cmd1Args[0] {
			case "echo":
				EchoCommand(cmd1Args)
				// Add other builtins here as needed
			}
			w.Close()
		}()
	} else {
		filePath1, exists1 := findBinInPath(cmd1Args[0])
		if !exists1 {
			return fmt.Errorf("%s: command not found", cmd1Args[0])
		}
		cmd1 := exec.Command(filePath1, cmd1Args[1:]...)
		cmd1.Stdout = w
		cmd1.Stderr = os.Stderr
		if err := cmd1.Start(); err != nil {
			w.Close()
			return err
		}
		go func() {
			cmd1.Wait()
			w.Close()
		}()
	}

	filePath2, exists2 := findBinInPath(cmd2Args[0])
	if !exists2 {
		return fmt.Errorf("%s: command not found", cmd2Args[0])
	}
	cmd2 := exec.Command(filePath2, cmd2Args[1:]...)
	cmd2.Stdin = r
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	return cmd2.Run()
}
