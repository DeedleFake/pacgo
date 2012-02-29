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
	"errors"
	"os"
)

func init() {
	RegisterCmd("-M", &Cmd{
		Help:      "Drop in replacement for makepkg with AUR support.",
		UsageLine: "-M [makepkg opts]",
		HelpMore: `-M scans a PKGBUILD for AUR dependencies, installs them, and then runs
makepkg with the given arguments. Note that, since it's supposed to be
a drop in replacement for makepkg, it will not install AUR
dependencies unless given the -s (or --syncdeps) flag.
`,
		Run: func(args ...string) error {
			file, err := os.Open("PKGBUILD")
			if err != nil {
				return err
			}
			defer file.Close()

			pb, err := ParsePkgbuild(file)
			if err != nil {
				return errors.New("Error parsing PKGBUILD: " + err.Error())
			}

			pkg, err := NewPkgbuildPkg(pb)
			if err != nil {
				return err
			}

			err = pkg.Install(nil, args[1:]...)
			if err != nil {
				return err
			}

			return nil
		},
	})
}
