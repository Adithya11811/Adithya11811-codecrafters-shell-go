package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
	// "github.com/google/shlex"
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

		cmd, argv := splitWithQuoting(strings.TrimSpace(input))
		// argv, err := shlex.Split(strings.TrimSpace(input))
		// cmd := argv[0]

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
			// ...existing code...
			parts := strings.Fields(input)
			if len(parts) == 0 {
				continue
			}

			if exists {
				var command *exec.Cmd
				if len(argv) == 0 {
					command = exec.Command(filePath)
					command.Args = []string{cmd}
				} else {
					command = exec.Command(filePath, argv[1:]...)
					command.Args = append([]string{cmd}, argv[1:]...)
				}
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
	if len(argv) < 2 {
		fmt.Fprintln(os.Stdout, "")
		return
	}
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

func splitWithQuoting(inputString string) (string, []string) {
	var current strings.Builder
	args := []string{}
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, c := range inputString {
		switch {
		case escaped:
			if inDoubleQuote {
				switch c {
				case '"', '\\', '$', '`':
					current.WriteRune(c)
				default:
					current.WriteRune('\\')
					current.WriteRune(c)
				}
			} else if !inSingleQuote {
				current.WriteRune(c)
			} else {
				current.WriteRune('\\')
				current.WriteRune(c)
			}
			escaped = false
		case c == '\\' && !inSingleQuote:
			escaped = true
		case c == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case c == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case unicode.IsSpace(c) && !inSingleQuote && !inDoubleQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(c)
		}
	}

	if escaped {
		current.WriteRune('\\')
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	if len(args) == 1 {
		return args[0], []string{}
	}
	return args[0], args
}
