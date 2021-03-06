package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type LineReader interface {
	ReadLine() ([]byte, bool, error)
}

// ReadLines reads from r, one line at a time, and returns the read
// lines as a [][]byte. If it encounters any errors, it returns nil
// and the error. It does not return io.EOF.
func ReadLines(r io.Reader, trim bool) ([][]byte, error) {
	var lr LineReader
	if rt, ok := r.(LineReader); ok {
		lr = rt
	} else {
		lr = bufio.NewReader(r)
	}

	var eof bool
	var lines [][]byte
	for !eof {
		var line []byte
		for {
			part, pre, err := lr.ReadLine()
			if err != nil {
				if err != io.EOF {
					return nil, err
				} else {
					eof = true
				}
			}
			line = append(line, part...)

			if !pre {
				break
			}
		}
		if trim {
			line = bytes.TrimSpace(line)
		}

		lines = append(lines, line)
	}

	return lines, nil
}

// ExtractTar extracts the contents of tr to the given dir. It
// returns an error, if any.
func ExtractTar(dir string, tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if hdr.Typeflag == tar.TypeDir {
			err = os.MkdirAll(filepath.Join(dir, hdr.Name), 0755)
			if err != nil {
				return err
			}
		} else {
			file, err := os.OpenFile(filepath.Join(dir, hdr.Name),
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.FileMode(hdr.Mode),
			)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// SplitArgs is a convience function that seperates pkgs from other
// arguments.
func SplitArgs(args ...string) (pacargs []string, pkgs []string) {
	for i, arg := range args {
		if arg == "--" {
			pkgs = append(pkgs, args[i+1:]...)
			break
		}

		if arg[0] == '-' {
			pacargs = append(pacargs, arg)
		} else {
			pkgs = append(pkgs, arg)
		}
	}

	return
}

// Copy of exp/terminal's IsTerminal() function.
func IsTerminal(fd int) bool {
	var t syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	return err == 0
}

// CheckConfOption checks if a given option is set in pacman.conf.
//
// TODO: Add support for configuration files that aren't located at
//				/etc/pacman.conf.
func CheckConfOption(opt string) (bool, error) {
	file, err := os.Open("/etc/pacman.conf")
	if err != nil {
		return false, err
	}
	defer file.Close()

	lines, err := ReadLines(file, true)
	if err != nil {
		return false, err
	}

	optS := []byte(opt)

	for _, line := range lines {
		if bytes.Equal(line, optS) {
			return true, nil
		}
	}

	return false, nil
}
