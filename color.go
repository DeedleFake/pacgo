package main

import (
	"bufio"
	"exp/terminal"
	"fmt"
	"os"
	"strings"
	"unicode"
)

var (
	Color1   string // Bold
	Color2   string // Green
	Color3   string // Purple
	Color4   string // Teal
	Color5   string // Blue
	Color6   string // Yellow
	Color7   string // Red
	ColorEnd string // None
)

// setColors sets up the colors. This is its own function so that it
// doesn't always get run.
func setColors() {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return
	}

	Color1 = "\033[1;39m"
	Color2 = "\033[1;32m"
	Color3 = "\033[1;35m"
	Color4 = "\033[1;36m"
	Color5 = "\033[1;34m"
	Color6 = "\033[1;33m"
	Color7 = "\033[1;31m"
	ColorEnd = "\033[0m"
}

// Colorize colorizes the given string using the following format:
//    [c1] -> Color1
//    [c2] -> Color2
//    ...
//    [ce] -> ColorEnd
//
// Returns the colorized string.
func Colorize(str string) string {
	str = strings.Replace(str, "[c1]", Color1, -1)
	str = strings.Replace(str, "[c2]", Color2, -1)
	str = strings.Replace(str, "[c3]", Color3, -1)
	str = strings.Replace(str, "[c4]", Color4, -1)
	str = strings.Replace(str, "[c5]", Color5, -1)
	str = strings.Replace(str, "[c6]", Color6, -1)
	str = strings.Replace(str, "[c7]", Color7, -1)
	str = strings.Replace(str, "[ce]", ColorEnd, -1)

	return str
}

// Cprintf colorizes and then print a string. This is a direct
// replacement for fmt.Printf().
func Cprintf(s string, args ...interface{}) (int, error) {
	return fmt.Print(Colorize(fmt.Sprintf(s, args...)))
}

// Caskf prints the given question, in color, appending the
// apporopriate question prompt to the end of it ([Y/n] or [y/N]). def
// is the default answer. It returns the result and nil, or false and
// an error, if any.
func Caskf(def bool, col string, s string, args ...interface{}) (bool, error) {
	q := fmt.Sprintf(" %v[y/N][ce] ", col)
	if def {
		q = fmt.Sprintf(" %v[Y/n][ce] ", col)
	}

	Cprintf(s+q, args...)

	bufin := bufio.NewReader(os.Stdin)
	c, err := bufin.ReadByte()
	if err != nil {
		return false, err
	}

	switch unicode.ToLower(rune(c)) {
	case 'y':
		def = true
	case 'n':
		def = false
	}

	return def, nil
}
