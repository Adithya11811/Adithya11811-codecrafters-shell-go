package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
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
	HistoryCommand(strings.Split(argv, " "), os.Stdin, os.Stdout, &hist)
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
func Menu(cmd string, argv []string) {
	switch cmd {
	case "exit":
		ExitCommand(argv, os.Stdin, os.Stdout, &hist)
	case "echo":
		EchoCommand(argv, os.Stdin, os.Stdout)
	case "type":
		TypeCommand(argv, os.Stdin, os.Stdout)
	case "pwd":
		getCurrentDir(argv, os.Stdin, os.Stdout)
	case "cd":
		if len(argv) < 2 {
			changeDir([]string{"cd", os.Getenv("HOME")}, os.Stdin, os.Stdout) // Default to HOME if no argument is provided
		} else {
			changeDir(argv, os.Stdin, os.Stdout)
		}
	case "history":
		HistoryCommand(argv, os.Stdin, os.Stdout, &hist)
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
			command.Stdin = os.Stdin
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

	if len(args) == 0 {
		return "", []string{}
	}

	return args[0], args
}
