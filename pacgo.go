package main

import (
	"fmt"
	"os"
)

type Cmd struct {
	Help string
	Run  func(...string) error
}

var cmds map[string]*Cmd

func RegisterCmd(arg string, cmd *Cmd) {
	if cmds == nil {
		cmds = make(map[string]*Cmd)
	}

	cmds[arg] = cmd
}

func Usage() {
	fmt.Printf("Usage: %v <cmd> [options]\n", os.Args[0])
	fmt.Println("Commands:")
	for name, cmd := range cmds {
		fmt.Printf("  %v: %v\n", name, cmd.Help)
	}
}

func main() {
	if len(os.Args) == 1 {
		Usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h", "-help", "--help":
		Usage()
		os.Exit(0)
	}

	if cmd, ok := cmds[os.Args[1]]; ok {
		err := cmd.Run(os.Args[2:]...)
		if err != nil {
			Cprintf("[c5]%v: [c7]error:[ce] %v\n", os.Args[1], err)
			os.Exit(1)
		}
	} else {
		Cprintf("[c7]error:[ce] No such command: [c5]%v[ce]\n", os.Args[1])
		Usage()
		os.Exit(2)
	}
}
