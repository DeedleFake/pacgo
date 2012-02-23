package main

import (
	"fmt"
	"os"
	"path/filepath"
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

var (
	TmpDir string
)

func MkTmpDir(name string) (string, error) {
	tmp := filepath.Join(TmpDir, name)
	err := os.MkdirAll(tmp, 0755)
	if err != nil {
		return tmp, fmt.Errorf("Failed to create %v.", tmp)
	}

	return tmp, nil
}

func Usage() {
	fmt.Printf("Usage: %v <cmd> [options]\n", os.Args[0])
	fmt.Println("Commands:")
	for name, cmd := range cmds {
		fmt.Printf("  %v: %v\n", name, cmd.Help)
	}
}

type UsageError struct {
	Arg string
}

func (err UsageError) Error() string {
	return fmt.Sprintf("Unknown argument: %v", err.Arg)
}

var (
	UpdateDevel bool
)

func main() {
	if os.Getuid() == 0 {
		Cprintf("[c7]error:[ce] Can't run as root.\n")
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		Usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h", "-help", "--help":
		Usage()
		os.Exit(0)
	}

	TmpDir = filepath.Join(os.TempDir(), fmt.Sprintf("%v-%v", filepath.Base(os.Args[0]), os.Getuid()))
	err := os.MkdirAll(TmpDir, 0755)
	if err != nil {
		Cprintf("[c7]error:[ce] Failed to create %v.", TmpDir)
		os.Exit(1)
	}

	if cmd, ok := cmds[os.Args[1]]; ok {
		err := cmd.Run(os.Args[1:]...)
		if err != nil {
			Cprintf("[c5]%v: [c7]error:[ce] %v\n", os.Args[1], err)
			if _, ok := err.(UsageError); ok {
				Usage()
				os.Exit(2)
			}
			os.Exit(1)
		}
	} else {
		Cprintf("[c7]error:[ce] No such command: [c5]%v[ce]\n", os.Args[1])
		Usage()
		os.Exit(2)
	}
}
