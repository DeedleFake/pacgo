package main

import (
	"errors"
	"fmt"
	"io/ioutil"
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
	err := os.Mkdir(tmp, 0755)
	if err != nil {
		return "", errors.New("Failed to create " + tmp)
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

	tmp, err := ioutil.TempDir("", filepath.Base(os.Args[0]))
	if err != nil {
		Cprintf("[c7]error:[ce] Failed to create %v. Does it already exist?", TmpDir)
		os.Exit(1)
	}
	TmpDir = tmp
	defer os.RemoveAll(TmpDir)

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
