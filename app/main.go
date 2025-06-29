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

	"github.com/cosiner/argv"
	// "golang.org/x/term"
	// "syscall"
	// "github.com/google/shlex"
)

var histFile string
var hist History
var trie *Trie

var builtIns = []string{"type", "echo", "exit", "pwd", "history"}

type History struct {
	Entries           []string
	lastAppendedIndex int
}

func (h *History) Add(entry string) {
	h.Entries = append(h.Entries, entry)
}

func (h *History) Get(index int) (string, bool) {
	if index >= 0 && index < len(h.Entries) {
		return h.Entries[index], true
	}
	return "", false
}

func (h *History) Len() int {
	return len(h.Entries)
}

func (h *History) WriteToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, entry := range h.Entries {
		fmt.Fprintln(file, entry)
	}
	return nil
}

func (h *History) AppendToFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for i := h.lastAppendedIndex; i < len(h.Entries); i++ {
		fmt.Fprintln(file, h.Entries[i])
	}
	h.lastAppendedIndex = len(h.Entries)
	return nil
}

func (h *History) ReadFromFile(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		if line != "" {
			h.Entries = append(h.Entries, line)
		}
	}
	h.lastAppendedIndex = len(h.Entries)
	return nil
}

func main() {
	hist = History{}
	histFile = os.Getenv("HISTFILE")
	argv := "history -r " + histFile
	HistoryCommand(strings.Split(argv, " "), &hist)
	trie = NewTrie()

	for i := 0; i < len(builtIns); i++ {
		trie.insert(builtIns[i])
	}

	for _, exe := range getPathExecutables() {
		trie.insert(exe)
	}

	for {
		// fmt.Fprint(os.Stdout, "$ ")
		historyIndex = hist.Len()
		input := handleInput("$ ")
		if len(input) == 0 {
			historyIndex = hist.Len() // Reset historyIndex after each input
			continue
		}
		hist.Add(input)
		historyIndex = hist.Len()
		trimmedInput := strings.TrimSpace(input)

		if strings.Contains(trimmedInput, "|") {
			HandlePipe(trimmedInput)
			continue
		}

		cmd, argv := splitWithQuoting(trimmedInput)
		// argv, err := shlex.Split(strings.TrimSpace(input))
		// cmd := argv[0]
		Menu(cmd, argv)

	}
}
func Menu(cmd string, argv[] string) {
			switch cmd {
		case "exit":
			ExitCommand(argv, &hist)
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
			HistoryCommand(argv, &hist)
			return
		default:
			filePath, exists := findBinInPath(cmd)
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

func ExitCommand(argv []string, hist *History) {
	code := 0

	if len(argv) > 1 {
		argCode, err := strconv.Atoi(argv[1])
		if err == nil { // Only set code if conversion is successful
			code = argCode
		}
	}

	temp := "history -w " + histFile
	HistoryCommand(strings.Split(temp, " "), hist)

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
		info, err := os.Stat(file)
		if err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
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

func HistoryCommand(argv []string, hist *History) {
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
						hist.Add(line)
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

			for i := 1; i <= hist.Len(); i++ {
				if command, exists := hist.Get(i - 1); exists {
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
		fmt.Fprintln(os.Stdout, "No commands in history.")
	} else {

		if cnt != 0 {
			cnt = hist.Len() - cnt
		}

		for i := cnt; i < hist.Len(); i++ {
			if command, exists := hist.Get(i); exists {
				fmt.Fprintf(os.Stdout, "    %d %s\n", i+1, command)
			}
		}
	}
}

func moveUpDownHistory(direction int) string {
	if hist.Len() == 0 {
		fmt.Fprintln(os.Stdout, "No commands in history.")
		return ""
	}
	cur_cnt := hist.Len() + 1

	if direction == 0 { // Move up
		if cur_cnt > 1 {
			cur_cnt--
		}
	} else if direction == 1 { // Move down
		if cur_cnt < len(hist.Entries) {
			cur_cnt++
		}
	}

	if command, exists := hist.Get(cur_cnt - 1); exists {
		fmt.Fprintf(os.Stdout, "%s", command)
		return command
	} else {
		fmt.Fprintln(os.Stdout, "No more commands in history.")
	}
	return ""
}

func getPathExecutables() []string {
	pathEnv := os.Getenv("PATH")
	paths := strings.Split(pathEnv, ":")
	seen := make(map[string]struct{})
	var executables []string

	for _, dir := range paths {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.Type().IsRegular() || file.Type()&os.ModeSymlink != 0 {
				name := file.Name()
				if _, ok := seen[name]; !ok {
					seen[name] = struct{}{}
					executables = append(executables, name)
				}
			}
		}
	}
	return executables
}
