package main

import (
	"errors"
	"os"
	"os/exec"
	"path"
)

var (
	PacmanPath  string
	MakepkgPath string

	SudoPath string
)

func init() {
	var err error

	PacmanPath, err = exec.LookPath("pacman-color")
	if err != nil {
		PacmanPath, err = exec.LookPath("pacman")
		if err != nil {
			Cprintf("[c7]error:[ce] Could not find pacman.\n")
			os.Exit(1)
		}
	} else {
		setColors()
	}

	MakepkgPath, err = exec.LookPath("makepkg")
	if err != nil {
		Cprintf("[c7]error:[ce] Could not find makepkg.\n")
		os.Exit(1)
	}

	SudoPath, err = exec.LookPath("sudo")
	if err != nil {
		Cprintf("[c6]warning:[ce] Could not find sudo.\n")
	}
}

func Pacman(args ...string) error {
	cmd := &exec.Cmd{
		Path: PacmanPath,
		Args: append([]string{path.Base(PacmanPath)}, args...),

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}

func SilentPacman(args ...string) error {
	cmd := &exec.Cmd{
		Path: PacmanPath,
		Args: append([]string{path.Base(PacmanPath)}, args...),
	}

	return cmd.Run()
}

func Makepkg(args ...string) error {
	cmd := &exec.Cmd{
		Path: MakepkgPath,
		Args: append([]string{path.Base(MakepkgPath)}, args...),

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}

func SudoPacman(args ...string) error {
	if SudoPath == "" {
		return errors.New("sudo not found.")
	}

	cmd := &exec.Cmd{
		Path: SudoPath,
		Args: append([]string{path.Base(SudoPath), PacmanPath}, args...),

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}
