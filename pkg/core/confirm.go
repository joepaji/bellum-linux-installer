package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Scanner is a wrapper around bufio.Scanner for reading user input
type Scanner struct {
	reader *bufio.Reader
}

// NewReader creates a new Scanner for reading from stdin
func NewReader() *Scanner {
	return &Scanner{reader: bufio.NewReader(os.Stdin)}
}

// ReadString reads a string terminated by a delimiter
func (s *Scanner) ReadString(delim byte) (string, error) {
	return s.reader.ReadString(delim)
}

// AskBool prompts the user for confirmation and returns true if they confirm
// If the user presses Enter without typing anything, it defaults to true (yes)
func AskBool(prompt string) bool {
	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return false
	}

	input = strings.TrimSpace(input)

	if input == "" {
		// Default to yes (empty input)
		return true
	}

	inputLower := strings.ToLower(input)
	if inputLower == "y" || inputLower == "yes" {
		return true
	}
	if inputLower == "n" || inputLower == "no" {
		return false
	}

	fmt.Printf("%sInvalid input: '%s'%s\n", ColorBoldRed, input, ColorReset)
	fmt.Println("Please enter 'y' or 'Y' to proceed, 'n' or 'N' to cancel, or press Enter to proceed (default: yes)")
	return AskBool(prompt)
}
