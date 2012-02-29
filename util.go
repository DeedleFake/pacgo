package main

import (
	"bufio"
	"bytes"
	"io"
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
