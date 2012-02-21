package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	PacmanCmd  *exec.Cmd
	MakepkgCmd *exec.Cmd

	SudoCmd *exec.Cmd
)

func init() {
	pac, err := exec.LookPath("pacman-color")
	if err != nil {
		pac, err = exec.LookPath("pacman")
		if err != nil {
			fmt.Println("Error: Could not find pacman.")
			os.Exit(1)
		}
	}

	PacmanCmd = &exec.Cmd{
		Path: pac,

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	mp, err := exec.LookPath("makepkg")
	if err != nil {
		fmt.Println("Error: Could not find makepkg.")
		os.Exit(1)
	}

	MakepkgCmd = &exec.Cmd{
		Path: mp,

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	sudo, err := exec.LookPath("sudo")
	if err != nil {
		fmt.Println("Warning: Could not find sudo.")
	}

	if sudo != "" {
		SudoCmd = &exec.Cmd{
			Path: sudo,

			Stdout: os.Stdout,
			Stdin:  os.Stdin,
			Stderr: os.Stderr,
		}
	}
}

func Pacman(args ...string) error {
	defer func() {
		PacmanCmd.Args = nil
	}()

	PacmanCmd.Args = append([]string{PacmanCmd.Path}, args...)

	return PacmanCmd.Run()
}

func Makepkg(args ...string) error {
	defer func() {
		MakepkgCmd.Args = nil
	}()

	MakepkgCmd.Args = append([]string{MakepkgCmd.Path}, args...)

	return MakepkgCmd.Run()
}

func SudoPacman(args ...string) error {
	if SudoCmd == nil {
		return errors.New("sudo not found.")
	}

	defer func() {
		SudoCmd.Args = nil
	}()

	SudoCmd.Args = append([]string{SudoCmd.Path, PacmanCmd.Path}, args...)

	return SudoCmd.Run()
}
