package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/exp/slices"
	"golang.org/x/term"
)

var historyIndex int // Add this at the top-level, outside any function

func handleInput(prompt string) string {
	var input strings.Builder
	// historyIndex is now a package-level variable, always set to hist.Len() before each input
	// historyIndex := hist.Len() // REMOVE this line

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to set raw mode:", err)
		os.Exit(1)
	}
	defer term.Restore(fd, oldState)

	fmt.Print(prompt)

	var lastTabInput string
	var tabCount int

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
			currInput := input.String()
			if currInput == lastTabInput {
				tabCount++
			} else {
				tabCount = 1
				lastTabInput = currInput
			}
			completions := trie.AutoComplete(currInput)
			if len(completions) == 0 {
				fmt.Print("\a") // Bell sound
				break
			}
			// Find the longest common prefix
			lcp := completions[0]
			for _, c := range completions[1:] {
				i := 0
				for i < len(lcp) && i < len(c) && lcp[i] == c[i] {
					i++
				}
				lcp = lcp[:i]
			}
			if len(lcp) > len(currInput) {
				toAdd := lcp[len(currInput):]
				input.WriteString(toAdd)
				fmt.Print(toAdd)
				// Add a space if there is exactly one match and the LCP is a full match
				if len(completions) == 1 && lcp == completions[0] {
					input.WriteString(" ")
					fmt.Print(" ")
				}
				tabCount = 0
				lastTabInput = ""
			} else {
				if tabCount == 1 {
					fmt.Print("\a") // First Tab: bell
				} else if tabCount == 2 {
					fmt.Print("\r\n")
					slices.Sort(completions)
					for i, c := range completions {
						if i > 0 {
							fmt.Print("  ")
						}
						fmt.Print(c)
					}
					fmt.Print("\r\n", prompt, currInput)
					tabCount = 0
					lastTabInput = ""
				}
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
					if historyIndex > 0 && hist.Len() > 0 {
						historyIndex--
						if cmd, ok := hist.Get(historyIndex); ok {
							input.Reset()
							input.WriteString(cmd)
							redrawInput(prompt, cmd) // Redraw prompt and recalled command
						}
					}
				case 'B': // Down arrow
					if historyIndex < hist.Len()-1 {
						historyIndex++
						if cmd, ok := hist.Get(historyIndex); ok {
							input.Reset()
							input.WriteString(cmd)
							redrawInput(prompt, cmd)
						}
					} else {
						historyIndex = hist.Len()
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
