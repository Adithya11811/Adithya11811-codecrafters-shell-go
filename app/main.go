package main

import (
	"bufio"
	"fmt"
	"os"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	// Uncomment this block to pass the first stage


	// Wait for user input
	for true{
		fmt.Fprint(os.Stdout, "$ ")
	command, err := bufio.NewReader(os.Stdin).ReadString('\n')

	if(err != nil) {
		fmt.Fprintln(os.Stderr, "Error reading command:", err)
		os.Exit(1)
	}
	fmt.Println(command[:len(command)-1] + ": command not found")
	}// Print the command without the trailing newline

}
