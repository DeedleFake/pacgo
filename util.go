package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
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
