package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func init() {
	RegisterCmd("-S", &Cmd{
		Help: "Install packages from a repo or the AUR.",
		Run: func(args ...string) error {
			pacargs, pkgs, err := ParseSyncArgs(args[1:]...)
			if err != nil {
				return err
			}

			return InstallPkgs(pacargs, pkgs)
		},
	})

	RegisterCmd("-Si", &Cmd{
		Help: "Get info about a remote package.",
		Run: func(args ...string) error {
			pacargs, pkgs, err := ParseSyncArgs(args[1:]...)
			if err != nil {
				return err
			}

			return InfoPkgs(pacargs, pkgs)
		},
	})

	RegisterCmd("-Ss", &Cmd{
		Help: "Search for keywords.",
		Run: func(args ...string) error {
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
					if e.ExitStatus() != 1 {
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

				Cprintf("[c3]aur/[c1]%v [c2]%v[ce]%v\n",
					info.GetSearch(i, "Name"),
					info.GetSearch(i, "Version"),
					<-ic,
				)
				Cprintf("    %v\n", info.GetSearch(i, "Description"))
			}

			return nil
		},
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
					if _, ok := InAUR(pkg.Name()); ok {
						up, err := Update(pkg)
						if err != nil {
							ac <- nil
							errc <- err
							return
						}
						if up != nil {
							aurpkgs = append(aurpkgs, up)
						}
					}
				}

				ac <- aurpkgs
				errc <- nil
			}()
		}

		if pkgs == nil {
			err := SudoPacman(append([]string{args[0]}, pacargs...)...)
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
		Help: "Install updates.",
		Run:  runUpdate,
	})

	RegisterCmd("-Syu", &Cmd{
		Help: "Update local package cache and install updates.",
		Run:  runUpdate,
	})

	RegisterCmd("-Scc", &Cmd{
		Help: "Clean leftover files.",
		Run: func(args ...string) error {
			if len(args) != 1 {
				return UsageError{args[1]}
			}

			err := SudoPacman(args...)
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
