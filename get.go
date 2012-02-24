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
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with pacgo. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

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
			err = os.Mkdir(filepath.Join(dir, hdr.Name), 0755)
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

func init() {
	RegisterCmd("-G", &Cmd{
		Help:      "Download the PKGBUILDs and other files for AUR packages.",
		UsageLine: "-G <pkgname>...",
		HelpMore: `-G downloads the source tarball for the given package(s) and extracts
them to the current directory. It accepts no arguments other than
package names, and will skip packages when it encounters errors.
`,
		Run: func(args ...string) error {
			if len(args) == 1 {
				return PrintUsageError
			}

			var pkgs []string
			for _, arg := range args[1:] {
				if arg[0] == '-' {
					return UsageError{arg}
				} else {
					pkgs = append(pkgs, arg)
				}
			}

			for _, pkg := range pkgs {
				tr, err := GetSourceTar(pkg)
				if err != nil {
					Cprintf("[c6]warning:[ce] Failed to get source tar for %v. Skipping...\n", pkg)
					continue
				}

				err = ExtractTar(".", tr)
				if err != nil {
					Cprintf("[c6]warning:[ce] Failed to extract %v. Skipping...", pkg)
					continue
				}
			}

			return nil
		},
	})
}
