package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	// "path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

var builtIns = []string{"type", "echo", "exit", "pwd"}

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")

		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

		argv := strings.Fields(input)
		cmd := argv[0]

		switch cmd {
		case "exit":
			ExitCommand(argv)
		case "echo":
			EchoCommand(argv)
		case "type":
			TypeCommand(argv)
		case "pwd":
			getCurrentDir()
		case "cd":
			if len(argv) < 2 {
				changeDir(os.Getenv("HOME")) // Default to HOME if no argument is provided
			} else {
				changeDir(argv[1])
			}
		default:
			filePath, exists := findBinInPath(cmd)
			if exists {
				command := exec.Command(filePath, argv[1:]...)

				command.Args = append([]string{cmd}, argv[1:]...)
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr
				if err := command.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %s\n", cmd, err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", cmd)
			}
		}
	}
}

func ExitCommand(argv []string) {
	code := 0

	if len(argv) > 1 {
		argCode, err := strconv.Atoi(argv[1])
		if err == nil { // Only set code if conversion is successful
			code = argCode
		}
	}

	os.Exit(code)
}

func EchoCommand(argv []string) {
	output := strings.Join(argv[1:], " ")
	fmt.Fprintf(os.Stdout, "%s\n", output)
}

func TypeCommand(argv []string) {
	if len(argv) == 1 {
		return
	}

	value := argv[1]

	if slices.Contains(builtIns, value) {
		fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", value)
		return
	}

	if file, exists := findBinInPath(value); exists {
		fmt.Fprintf(os.Stdout, "%s is %s\n", value, file)
		return
	}

	fmt.Fprintf(os.Stdout, "%s: not found\n", value)
}

func findBinInPath(bin string) (string, bool) {
	paths := os.Getenv("PATH")
	for _, path := range strings.Split(paths, ":") {
		file := filepath.Join(path, bin)
		if _, err := os.Stat(file); err == nil {
			return file, true
		}
	}

	return "", false
}

func getCurrentDir() {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %s\n", err)

	}
	fmt.Fprintf(os.Stdout, "%s\n", currentDir)
}

func changeDir(path string) {
	if path == "~" || path == "$HOME" {
		path = os.Getenv("HOME")
	}
	if err := os.Chdir(path); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", path)
	}
}
