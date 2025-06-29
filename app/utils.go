package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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

func HandlePipe(input string) {
	parts := strings.SplitN(input, "|", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	err := executePipelineBuiltinAware(strings.Split(left, " "), strings.Split(right, " "))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing pipeline: %s\n", err)
		return
	}
}


// Pipeline handler that supports builtins on both sides
func executePipelineBuiltinAware(cmd1Args []string, cmd2Args []string) error {
	r, w := io.Pipe()

	// Left side
	leftIsBuiltin := isBuiltin(cmd1Args[0])
	// Right side
	rightIsBuiltin := isBuiltin(cmd2Args[0])

	errChan := make(chan error, 2)

	// LEFT
	go func() {
		if leftIsBuiltin {
			// Call builtin with w as output
			callBuiltin(cmd1Args, os.Stdin, w)
			w.Close()
			errChan <- nil
		} else {
			filePath1, exists1 := findBinInPath(cmd1Args[0])
			if !exists1 {
				w.Close()
				errChan <- fmt.Errorf("%s: command not found", cmd1Args[0])
				return
			}
			cmd1 := exec.Command(filePath1, cmd1Args[1:]...)
			cmd1.Stdout = w
			cmd1.Stderr = os.Stderr
			cmd1.Stdin = os.Stdin
			err := cmd1.Run()
			w.Close()
			errChan <- err
		}
	}()

	// RIGHT
	if rightIsBuiltin {
		callBuiltin(cmd2Args, r, os.Stdout)
		io.Copy(io.Discard, r) // Drain the pipe to avoid deadlock
		return <-errChan
	} else {
		filePath2, exists2 := findBinInPath(cmd2Args[0])
		if !exists2 {
			return fmt.Errorf("%s: command not found", cmd2Args[0])
		}
		cmd2 := exec.Command(filePath2, cmd2Args[1:]...)
		cmd2.Stdin = r
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		err := cmd2.Run()
		return err
	}
}

// Helper to call a builtin by name and argv
func callBuiltin(argv []string, in io.Reader, out io.Writer) {
	switch argv[0] {
	case "exit":
		ExitCommand(argv, in, out, &hist)
	case "echo":
		EchoCommand(argv, in, out)
	case "type":
		TypeCommand(argv, in, out)
	case "pwd":
		getCurrentDir(argv, in, out)
	case "cd":
		changeDir(argv, in, out)
	case "history":
		HistoryCommand(argv, in, out, &hist)
	}
}

func ExitCommand(argv []string, in io.Reader, out io.Writer, hist *History) {
	code := 0
	if len(argv) > 1 {
		argCode, err := strconv.Atoi(argv[1])
		if err == nil {
			code = argCode
		}
	}
	temp := "history -w " + histFile
	HistoryCommand(strings.Split(temp, " "), in, out, hist)
	os.Exit(code)
}

func EchoCommand(argv []string, in io.Reader, out io.Writer) {
	if len(argv) < 2 {
		fmt.Fprintln(out, "")
		return
	}
	output := strings.Join(argv[1:], " ")
	fmt.Fprintf(out, "%s\n", output)
}

func TypeCommand(argv []string, in io.Reader, out io.Writer) {
	if len(argv) == 1 {
		return
	}
	value := argv[1]
	if slices.Contains(builtIns, value) {
		fmt.Fprintf(out, "%s is a shell builtin\n", value)
		return
	}
	if file, exists := findBinInPath(value); exists {
		fmt.Fprintf(out, "%s is %s\n", value, file)
		return
	}
	fmt.Fprintf(out, "%s: not found\n", value)
}

func getCurrentDir(argv []string, in io.Reader, out io.Writer) {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(out, "Error getting current directory: %s\n", err)
	}
	fmt.Fprintf(out, "%s\n", currentDir)
}

func changeDir(argv []string, in io.Reader, out io.Writer) {
	if len(argv) < 2 {
		fmt.Fprintf(out, "cd: missing argument\n")
		return
	}
	path := argv[1]
	if path == "~" || path == "$HOME" {
		path = os.Getenv("HOME")
	}
	if err := os.Chdir(path); err != nil {
		fmt.Fprintf(out, "cd: %s: No such file or directory\n", path)
	}
}

func HistoryCommand(argv []string, in io.Reader, out io.Writer, hist *History) {
	cnt := 0
	if len(argv) > 1 {
		argCode, err := strconv.Atoi(argv[1])
		if err == nil {
			cnt = argCode
		}
	}
	if len(argv) > 2 {
		if argv[1] == "-r" {
			if _, err := os.Stat(argv[2]); err == nil && !os.IsNotExist(err) {
				file, err := os.ReadFile(argv[2])
				if err != nil {
					fmt.Fprintf(out, "Error reading history file: %s\n", err)
					return
				}
				lines := strings.Split(string(file), "\n")
				for _, line := range lines {
					if line != "" {
						hist.Add(line)
					}
				}
				return
			} else {
				return
			}
		}
		if argv[1] == "-w" {
			if len(argv) < 3 {
				fmt.Fprintln(out, "Usage: history -w <filename>")
				return
			}
			file, err := os.Create(argv[2])
			if err != nil {
				return
			}
			defer file.Close()
			for i := 1; i <= hist.Len(); i++ {
				if command, exists := hist.Get(i - 1); exists {
					fmt.Fprintf(file, "%s\n", command)
				}
			}
			return
		}
		if argv[1] == "-a" {
			if len(argv) < 3 {
				fmt.Fprintln(out, "Usage: history -a <filename>")
				return
			}
			file, err := os.OpenFile(argv[2], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(out, "Error opening history file: %s\n", err)
				return
			}
			defer file.Close()
			for i := hist.lastAppendedIndex; i < hist.Len(); i++ {
				if command, exists := hist.Get(i); exists {
					fmt.Fprintln(file, command)
				}
			}
			hist.lastAppendedIndex = hist.Len()
			return
		}
	}
	if hist.Len() == 0 {
		fmt.Fprintln(out, "No commands in history.")
	} else {
		if cnt != 0 {
			cnt = hist.Len() - cnt
		}
		for i := cnt; i < hist.Len(); i++ {
			if command, exists := hist.Get(i); exists {
				fmt.Fprintf(out, "    %d %s\n", i+1, command)
			}
		}
	}
}

func findBinInPath(bin string) (string, bool) {
	paths := os.Getenv("PATH")
	for _, path := range strings.Split(paths, ":") {
		file := filepath.Join(path, bin)
		info, err := os.Stat(file)
		if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return file, true
		}
	}
	return "", false
}
