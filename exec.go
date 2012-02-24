package main

import (
	"errors"
	"os"
	"os/exec"
	"path"
)

// These store the paths to the various executables that are run by
// pacgo.
var (
	PacmanPath  string
	MakepkgPath string
	VercmpPath  string

	SudoPath string

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

	// Find sudo.
	SudoPath, err = exec.LookPath("sudo")
	if err != nil {
		Cprintf("[c6]warning:[ce] Could not find sudo.\n")
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

// SudoPacman runs sudo pacman, passing the given args to it. It
// returns an error, if any.
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
