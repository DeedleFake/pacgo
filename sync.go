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
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func init() {
	RegisterCmd("-S", &Cmd{
		Help:      "Install packages from a repo or the AUR.",
		UsageLine: "-S [pacman opts] <packages>",
		HelpMore: `-S installs the listes packages, first getting everything it can from
pacman, and then installing any packages it can find in the AUR. It
will fail if it can't find a package. All options are passed straight
through to pacman.
`,
		Run: func(args ...string) error {
			pacargs, pkgs, err := ParseSyncArgs(args[1:]...)
			if err != nil {
				return err
			}

			return InstallPkgs(pacargs, pkgs)
		},
	})

	RegisterCmd("-Si", &Cmd{
		Help:      "Get info about a remote package.",
		UsageLine: "-Si [pacman opts] <packages>",
		HelpMore: `-Si prints information about the given packages, including packages
in the AUR. Unlike pacman, it will fail if given no arguments.
`,
		Run: func(args ...string) error {
			if len(args) == 1 {
				return PrintUsageError
			}

			pacargs, pkgs, err := ParseSyncArgs(args[1:]...)
			if err != nil {
				return err
			}

			return InfoPkgs(pacargs, pkgs)
		},
	})

	runSearch := func(args ...string) error {
		if len(args) == 1 {
			return PrintUsageError
		}

		for _, arg := range args {
			switch arg {
			case "-q", "--quiet":
				args[0] = "-Ssq"
			}
		}

		sc := make(chan RPCResult)
		errc := make(chan error)
		go func() {
			var search []string
			for _, arg := range args[1:] {
				if arg[0] != '-' {
					search = append(search, arg)
				}
			}

			info, err := AURSearch(strings.Join(search, " "))
			sc <- info
			errc <- err
		}()

		err := Pacman(args...)
		if err != nil {
			switch e := err.(type) {
			case *exec.ExitError:
				if ws, ok := e.Sys().(syscall.WaitStatus); ok && ws.ExitStatus() != 1 {
					<-sc
					<-errc
					return err
				}
			default:
				<-sc
				<-errc
				return err
			}
		}

		info := <-sc
		err = <-errc
		if err != nil {
			if err.Error() == "No results found" {
				return nil
			}
			return err
		}

		ic := make(chan string)
		for i := range info.Results.([]interface{}) {
			go func() {
				installed := ""
				if InLocal(info.GetSearch(i, "Name")) {
					installed = " [c4][installed][ce]"
				}

				ic <- installed
			}()

			if args[0] != "-Ssq" {
				Cprintf("[c3]aur/[c1]%v [c2]%v[ce]%v\n",
					info.GetSearch(i, "Name"),
					info.GetSearch(i, "Version"),
					<-ic,
				)
				Cprintf("    %v\n", info.GetSearch(i, "Description"))
			} else {
				Cprintf("%v\n", info.GetSearch(i, "Name"))
			}
		}

		return nil
	}

	RegisterCmd("-Ss", &Cmd{
		Help:      "List packages matching keywords.",
		UsageLine: "-Ss [pacman opts] <keywords>",
		HelpMore: `-Ss prints a list of packages matching the keywords, the repo they're
in, their version, and and an indicator that they're installed if
they're installed. Unlike pacman, it will fail if given no arguments.
`,
		Run: runSearch,
	})

	RegisterCmd("-Ssq", &Cmd{
		Help:      "List the names of packages matching keywords.",
		UsageLine: "-Ssq [pacman opts] <keywords>",
		HelpMore: `-Ssq prints a list of packages matching the keywords. Unlike -Ss, it
only lists their names. Like -Ss, it will fail if given no arguments.
`,
		Run: runSearch,
	})

	runUpdate := func(args ...string) error {
		pacargs, pkgs, err := ParseSyncArgs(args[1:]...)
		if err != nil {
			return err
		}

		if pkgs != nil {
			return errors.New("Using " + args[0] + " with specific packages is not yet supported.")
		}

		ac := make(chan []Pkg)
		errc := make(chan error)
		if pkgs == nil {
			go func() {
				pkgs, err := ListLocalPkgs()
				if err != nil {
					ac <- nil
					errc <- err
					return
				}

				var aurpkgs []Pkg
				for _, pkg := range pkgs {
					if info, ok := InAUR(pkg.Name()); ok {
						aurpkg, err := NewAURPkg(info)
						if err != nil {
							ac <- nil
							errc <- err
							return
						}
						if (UpdateVCS) && (aurpkg.IsVCS()) {
							aurpkgs = append(aurpkgs, aurpkg)
						} else {
							ver1, err := aurpkg.Version()
							if err != nil {
								ac <- nil
								errc <- err
								return
							}
							ver2, err := pkg.Version()
							if err != nil {
								ac <- nil
								errc <- err
								return
							}
							if up, err := Newer(ver1, ver2); (err == nil) && up {
								aurpkgs = append(aurpkgs, aurpkg)
							} else if err != nil {
								ac <- nil
								errc <- err
								return
							}
						}
					}
				}

				ac <- aurpkgs
				errc <- nil
			}()
		}

		if pkgs == nil {
			err := AsRootPacman(append([]string{args[0]}, pacargs...)...)
			if err != nil {
				<-ac
				<-errc
				return err
			}
		} else {
			err := InstallPkgs(args[1:], pkgs)
			if err != nil {
				return err
			}
		}

		if pkgs == nil {
			fmt.Println()
			Cprintf("[c5]:: [c1]Calculating AUR updates...[ce]\n")

			aurpkgs := <-ac
			err = <-errc
			if err != nil {
				return err
			}

			if aurpkgs == nil {
				Cprintf(" there is nothing to do\n")
			}

			fmt.Println()
			Cprintf("[c6]Targets (%v):[ce]", len(aurpkgs))
			for _, pkg := range aurpkgs {
				Cprintf(" %v", pkg.Name())
			}
			Cprintf("\n\n")
			answer, err := Caskf(true, "", "Proceed with installation?")
			if err != nil {
				return err
			}
			if !answer {
				return nil
			}

			for _, pkg := range aurpkgs {
				if ip, ok := pkg.(InstallPkg); ok {
					isdep, err := IsDep(pkg.Name())
					if err != nil {
						return err
					}
					asdeps := ""
					if isdep {
						asdeps = "--asdeps"
					}

					err = ip.Install(nil, asdeps)
					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("Don't know how to install %v.", pkg.Name())
				}
			}
		}

		return nil
	}

	RegisterCmd("-Su", &Cmd{
		Help:      "Install updates.",
		UsageLine: "-Su [opts]",
		HelpMore: `-Su checks for updates to all installed, pacman and AUR, packages and
downloads and installs them.

-Su also takes these non-pacman options:
	--upvcs: Update VCS AUR packages.

It is not yet capable of updating specific packages, but this
functionality is intended.

See also: -Syu
`,
		Run: runUpdate,
	})

	RegisterCmd("-Syu", &Cmd{
		Help:      "Update local package cache and install updates.",
		UsageLine: "-Syu [pacman opts]",
		HelpMore: `-Syu is exactly like -Su, but it also updates the local pacman
package databases. AUR updates are not affected.

See also: -Su
`,
		Run: runUpdate,
	})

	RegisterCmd("-Scc", &Cmd{
		Help:      "Clean leftover files.",
		UsageLine: "-Scc",
		HelpMore: `-Scc is a convience command that runs pacman -Scc and then gives the
option to remove pacgo's temporary directory. Unlike pacman, it
accepts no arguments.
`,
		Run: func(args ...string) error {
			if len(args) != 1 {
				return UsageError{args[1]}
			}

			err := AsRootPacman(args...)
			if err != nil {
				return err
			}

			fmt.Println()
			Cprintf("[c1]TmpDir:[ce] %v\n", TmpDir)
			answer, err := Caskf(false, "", "Do you want to remove TmpDir?")
			if err != nil {
				return err
			}
			if answer {
				Cprintf("removing TmpDir...\n")
				err = os.RemoveAll(TmpDir)
				if err != nil {
					return err
				}
			}

			return nil
		},
	})
}

// ParseSyncArgs is a convience function that seperates pkgs from
// other arguments. If it encounters an error, it returns nil, nil,
// and the error.
func ParseSyncArgs(args ...string) (pacargs []string, pkgs []Pkg, err error) {
	for _, arg := range args {
		if arg[0] == '-' {
			pacargs = append(pacargs, arg)
		} else {
			pkg, err := NewRemotePkg(arg)
			if err != nil {
				return nil, nil, err
			}
			pkgs = append(pkgs, pkg)
		}
	}

	return
}
