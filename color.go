package main

import (
	"exp/terminal"
	"fmt"
	"os"
	"strings"
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

func Cprintf(s string, args ...interface{}) (int, error) {
	return fmt.Print(Colorize(fmt.Sprintf(s, args...)))
}
