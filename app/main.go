package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

var Keywords = map[string]bool{
	"exit": true,
}

func main() {
	run()
}


func run() {
		// Wait for user input
	for true{
		fmt.Fprint(os.Stdout, "$ ")
	command, err := bufio.NewReader(os.Stdin).ReadString('\n')

	if(err != nil) {
		fmt.Fprintln(os.Stderr, "Error reading command:", err)
		os.Exit(1)
	}

	commands := strings.Fields(command)
    if len(commands) == 2 {
        if Keywords[commands[0]] && commands[1] == "0" {
            os.Exit(0)
        }
    }

	fmt.Println(strings.TrimSpace(command) + ": command not found")
	}
}
