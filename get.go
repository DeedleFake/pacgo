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
	"sync"
)

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

			var wg sync.WaitGroup
			for _, pkg := range pkgs {
				wg.Add(1)
				go func(pkg string) {
					defer wg.Done()

					tr, err := GetSourceTar(pkg)
					if err != nil {
						Cprintf("[c6]warning:[ce] Failed to get source tar for %v. Skipping...\n", pkg)
						return
					}

					err = ExtractTar(".", tr)
					if err != nil {
						Cprintf("[c6]warning:[ce] Failed to extract %v. Skipping...\n", pkg)
						return
					}
				}(pkg)
			}

			wg.Wait()

			return nil
		},
	})
}
