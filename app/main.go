package main

import (
	// "bufio"
	// "bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	// "syscall"
	"unicode"
	// "golang.org/x/term"
	// "syscall"
	// "github.com/google/shlex"
)

var hist_map = make(map[int]string)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

var builtIns = []string{"type", "echo", "exit", "pwd", "history"}
var hist_cnt int = 0
var lastAppendedHistoryIndex int
var histFile string

func main() {
	histFile = os.Getenv("HISTFILE")
	argv := "history -r " + histFile
	HistoryCommand(strings.Split(argv, " "))
	for {
		// fmt.Fprint(os.Stdout, "$ ")

		input := handleInput("$ ")
		// os.Stdout.Write([]byte("strawberry blueberry"))

		// input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		// if err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
		// 	continue
		// }
		// fmt.Print(len(input))
		if len(input) == 0 { // this means just enter key is pressed
			continue
		}
		hist_cnt++

		hist_map[hist_cnt] = input

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
		case "history":
			HistoryCommand(argv)
			continue
		default:
			filePath, exists := findBinInPath(cmd)

			// parts := strings.Fields(input)
			// if len(parts) == 0 {
			// 	continue
			// }

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

	temp := "history -w " + histFile
	HistoryCommand(strings.Split(temp, " "))

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

	// ✅ Edge case fix
	if len(args) == 0 {
		return "", []string{}
	}

	return args[0], args
}

func HistoryCommand(argv []string) {
	cnt := 0

	if len(argv) > 1 {
		argCode, err := strconv.Atoi(argv[1])
		if err == nil { // Only set code if conversion is successful
			cnt = argCode
		}
	}
	// fmt.Printf("%v %T\n", argv, argv)
	if len(argv) > 2 {
		if argv[1] == "-r" {
			//check if argv[2] file exists
			if _, err := os.Stat(argv[2]); err == nil && !os.IsNotExist(err) {
				file, err := os.ReadFile(argv[2])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading history file: %s\n", err)
					return
				}

				lines := strings.Split(string(file), "\n")
				for _, line := range lines {
					if line != "" {
						hist_cnt++
						hist_map[hist_cnt] = line
					}
				}
				return // <-- Only load history, do not print anything
			} else {
				// fmt.Fprintf(os.Stderr, "History file %s does not exist.\n", argv[2])
				return
			}
		}
		if argv[1] == "-w" {
			if len(argv) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: history -w <filename>")
				return
			}
			file, err := os.Create(argv[2])
			if err != nil {
				return
			}
			defer file.Close()

			for i := 1; i <= hist_cnt; i++ {
				if command, exists := hist_map[i]; exists {
					fmt.Fprintf(file, "%s\n", command)
				}
			}

			return
		}

		if argv[1] == "-a" {
			if len(argv) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: history -a <filename>")
				return
			}
			file, err := os.OpenFile(argv[2], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening history file: %s\n", err)
				return
			}
			defer file.Close()

			for i := lastAppendedHistoryIndex + 1; i <= hist_cnt; i++ {
				if command, exists := hist_map[i]; exists {
					fmt.Fprintln(file, command)
				}
			}
			lastAppendedHistoryIndex = hist_cnt
			return

		}
	}

	if hist_cnt == 0 {
		fmt.Fprintln(os.Stdout, "No commands in history.")
	} else {

		if cnt != 0 {
			cnt = hist_cnt - cnt
		}

		for i := cnt; i < hist_cnt; i++ {
			if command, exists := hist_map[i+1]; exists {
				fmt.Fprintf(os.Stdout, "    %d %s\n", i+1, command)
			}
		}
	}
}

func moveUpDownHistory(direction int) string {
	if hist_cnt == 0 {
		fmt.Fprintln(os.Stdout, "No commands in history.")
		return ""
	}
	cur_cnt := hist_cnt + 1

	if direction == 0 { // Move up
		if cur_cnt > 1 {
			cur_cnt--
		}
	} else if direction == 1 { // Move down
		if cur_cnt < len(hist_map) {
			cur_cnt++
		}
	}

	if command, exists := hist_map[cur_cnt]; exists {
		fmt.Fprintf(os.Stdout, "%s", command)
		return command
	} else {
		fmt.Fprintln(os.Stdout, "No more commands in history.")
	}
	return ""
}
