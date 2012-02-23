package main

import (
	"os"
)

func init() {
	RegisterCmd("-M", &Cmd{
		Help: "Read a local PKGBUILD, install AUR dependencies, and run makepkg.",
		Run: func(args ...string) error {
			file, err := os.Open("PKGBUILD")
			if err != nil {
				return err
			}
			defer file.Close()

			pb, err := ParsePkgbuild(file)
			if err != nil {
				return err
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
