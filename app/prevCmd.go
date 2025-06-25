package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func handleInput1() string {
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to set raw mode:", err)
		return ""
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	termIn := os.NewFile(uintptr(syscall.Stdin), "/dev/stdin")
	// t := term.NewTerminal(termIn, "")

	var line []rune
	historyIndex := hist_cnt + 1

	for {
		b := make([]byte, 1)
		_, err := termIn.Read(b)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Read error:", err)
			return ""
		}

		switch b[0] {
		case 13: // Enter
			// fmt.Print("\r\n")
			return string(line)

		case 127: // Backspace
			if len(line) > 0 {
				line = line[:len(line)-1]
				fmt.Print("\r\x1b[K$ ", string(line))
			}

		case 27: // Escape sequence (arrow keys)
			seq := make([]byte, 2)
			_, err := termIn.Read(seq)
			if err != nil {
				continue
			}
			if seq[0] == 91 {
				switch seq[1] {
				case 65: // Up
					if historyIndex > 1 {
						historyIndex--
						if cmd, ok := hist_map[historyIndex]; ok {
							line = []rune(cmd)
							fmt.Print("\r\x1b[2K") // Clear full line
							fmt.Print("$ ", cmd)   // Redraw prompt and command
						}
					}
				case 66: // Down
					if historyIndex < hist_cnt {
						historyIndex++
						if cmd, ok := hist_map[historyIndex]; ok {
							line = []rune(cmd)
							fmt.Print("\r\x1b[K$ ", string(line))
						}
					} else {
						historyIndex = hist_cnt + 1
						line = []rune{}
						fmt.Print("\r\x1b[K$ ")
					}
				}
			}

		default:
			if b[0] >= 32 && b[0] <= 126 {
				line = append(line, rune(b[0]))
				fmt.Printf("%c", b[0])
			}
		}
	}
}

func handleInput(prompt string) string {
	var input strings.Builder
	historyIndex := hist_cnt + 1 // Start just after the latest entry

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to set raw mode:", err)
		os.Exit(1)
	}
	defer term.Restore(fd, oldState)

	fmt.Print(prompt)

	for {
		var buf [1]byte
		n, err := os.Stdin.Read(buf[:])
		if err != nil || n == 0 {
			fmt.Fprintln(os.Stderr, "Read error:", err)
			return ""
		}

		char := buf[0]

		switch char {
		case '\n', '\r': // Enter key
			fmt.Print("\r\n")
			return input.String()

		case 127, 8: // Backspace
			if input.Len() > 0 {
				curr := input.String()
				input.Reset()
				input.WriteString(curr[:len(curr)-1])
				fmt.Print("\b \b") // Move back, overwrite with space, move back again
			}

		case 3: // Ctrl+C
			fmt.Print("\n")
			term.Restore(fd, oldState)
			os.Exit(0)

		case 9: // Tab key
			completions := trie.AutoComplete(input.String())
			if len(completions) == 0 {
				// No completions, do nothing
				break
			}
			if len(completions) == 1 {
				// Only one completion, auto-complete the input and add a space
				curr := input.String()
				toAdd := completions[0][len(curr):] + " "
				input.WriteString(toAdd)
				fmt.Print(toAdd)
			} else {
				// Multiple completions, print them all
				fmt.Print("\r\n")
				for _, c := range completions {
					fmt.Println(c)
				}
				// Redraw the prompt and current input
				fmt.Print(prompt, input.String())
			}
			continue

		case 27: // Escape sequence (e.g. arrow keys)
			var buf2 [2]byte
			n, err := os.Stdin.Read(buf2[:])
			if err != nil || n < 2 {
				continue
			}

			if buf2[0] == '[' {
				switch buf2[1] {
				case 'A': // Up arrow
					if historyIndex > 1 {
						historyIndex--
						if cmd, ok := hist_map[historyIndex]; ok {
							input.Reset()
							input.WriteString(cmd)
							redrawInput(prompt, cmd)
						}
					}
				case 'B': // Down arrow
					if historyIndex < hist_cnt {
						historyIndex++
						if cmd, ok := hist_map[historyIndex]; ok {
							input.Reset()
							input.WriteString(cmd)
							redrawInput(prompt, cmd)
						}
					} else {
						historyIndex = hist_cnt + 1
						input.Reset()
						redrawInput(prompt, "")
					}
				}
			}

		default:
			// Printable ASCII characters
			if char >= 32 && char < 127 {
				input.WriteByte(char)
				fmt.Print(string(char))
			}
		}
	}
}

func redrawInput(prompt, content string) {
	fmt.Print("\r\033[K")               // Clear line
	fmt.Printf("%s%s", prompt, content) // Print prompt and current input
}
