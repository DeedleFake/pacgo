// Copyright 2012 Yissakhar Z. Beck
//
// This file is part of pacgo.
// 
// pacgo is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// pacgo is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with pacgo. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
)

// These store the paths to the various executables that are run by
// pacgo.
var (
	PacmanPath  string
	MakepkgPath string
	VercmpPath  string

	// Using sudo?
	Sudo       bool
	AsRootPath string

	EditPath string

	BashPath string
)

func init() {
	var err error

	// Find pacman, trying pacman-color first, and then falling
	// back to normal pacman.
	PacmanPath, err = exec.LookPath("pacman-color")
	if err != nil {
		PacmanPath, err = exec.LookPath("pacman")
		if err != nil {
			Cprintf("[c7]error:[ce] Could not find pacman.\n")
			os.Exit(1)
		}
	} else {
		// Only set the colors if pacman will have colored output.
		setColors()
	}

	// Find makepkg.
	MakepkgPath, err = exec.LookPath("makepkg")
	if err != nil {
		Cprintf("[c7]error:[ce] Could not find makepkg.\n")
		os.Exit(1)
	}

	// Find vercmp.
	VercmpPath, err = exec.LookPath("vercmp")
	if err != nil {
		Cprintf("[c7]error:[ce] Could not find vercmp.\n")
		os.Exit(1)
	}

	// Find sudo. If you can't find it, use su.
	AsRootPath, err = exec.LookPath("sudo")
	if err != nil {
		AsRootPath, err = exec.LookPath("su")
		if err != nil {
			Cprintf("[c6]warning:[ce] Could not find sudo or su.\n")
		} else {
			Cprintf("[c6]warning:[ce] Could not find sudo. Using su.\n")
		}
	} else {
		Sudo = true
	}

	// Find the editor. Try the $EDITOR environment variable first.
	// If it's not set, use vim. If you can't find $EDITOR or vim, try
	// nano. If you still can't find it, warn about it.
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	EditPath, err = exec.LookPath(editor)
	if err != nil {
		EditPath, err = exec.LookPath("nano")
		if err != nil {
			Cprintf("[c6]warning:[ce] Could not find %v or nano.\n", editor)
		}
	}

	// Find bash.
	BashPath, err = exec.LookPath("bash")
	if err != nil {
		Cprintf("[c7]error:[ce] Could not find bash.\n")
	}
}

// Pacman runs pacman, passing the given argus to it. It returns an
// error, if any.
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

// SilentPacman runs pacman, passing it the given args, but unlike
// Pacman(), it does not give it access to stdout, stdin, and stderr.
// It returns an error, if any.
func SilentPacman(args ...string) error {
	cmd := &exec.Cmd{
		Path: PacmanPath,
		Args: append([]string{path.Base(PacmanPath)}, args...),
	}

	return cmd.Run()
}

// PacmanOutput runs pacman, passing the given args to it, and returns
// its output and an error, if any.
func PacmanOutput(args ...string) ([]byte, error) {
	cmd := &exec.Cmd{
		Path: PacmanPath,
		Args: append([]string{path.Base(PacmanPath)}, args...),
	}

	return cmd.Output()
}

// PacmanLines returns a [][]byte containing the lines output by
// running pacman with the given args. trim is passed through to
// ReadLines(). If it encounters any errors, it returns nil and the
// error.
func PacmanLines(trim bool, args ...string) ([][]byte, error) {
	cmd := &exec.Cmd{
		Path: PacmanPath,
		Args: append([]string{path.Base(PacmanPath)}, args...),
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	lines, err := ReadLines(out, trim)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return lines, nil
}

// MakepkgIn runs makepkg in the given dir, passing the given args to
// it. It returns an error, if any.
func MakepkgIn(dir string, args ...string) error {
	cmd := &exec.Cmd{
		Path: MakepkgPath,
		Args: append([]string{path.Base(MakepkgPath)}, args...),
		Dir:  dir,

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}

// VercmpOutput runs vercmp, passing the given args to it. It returns
// its output and an error, if any.
func VercmpOutput(args ...string) ([]byte, error) {
	cmd := &exec.Cmd{
		Path: VercmpPath,
		Args: append([]string{path.Base(VercmpPath)}, args...),
	}

	return cmd.Output()
}

// AsRootPacman runs pacman as root, passing the given args to it. It
// returns an error, if any.
func AsRootPacman(args ...string) error {
	if AsRootPath == "" {
		return errors.New("Could not find sudo or su.")
	}

	args = append([]string{PacmanPath}, args...)

	var cmdargs []string
	if Sudo {
		cmdargs = make([]string, 0, len(args)+1)
		cmdargs = append(cmdargs, path.Base(AsRootPath))
		cmdargs = append(cmdargs, args...)
	} else {
		cmdargs = make([]string, 0, 3)
		cmdargs = append(cmdargs, path.Base(AsRootPath), "-c")
		cmdargs = append(cmdargs, strings.Join(args, " "))
	}

	cmd := &exec.Cmd{
		Path: AsRootPath,
		Args: cmdargs,

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	if !Sudo {
		Cprintf("Root ")
	}
	return cmd.Run()
}

// Edit runs the editor, passing the given args to it. It returns an
// error, if any.
func Edit(args ...string) error {
	if EditPath == "" {
		panic("This should never happen.")
	}

	cmd := &exec.Cmd{
		Path: EditPath,
		Args: append([]string{path.Base(EditPath)}, args...),

		Stdout: os.Stdout,
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}
