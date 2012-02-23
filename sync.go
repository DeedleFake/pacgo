package main

import (
	"os/exec"
	"strings"
)

func init() {
	RegisterCmd("-S", &Cmd{
		Help: "Install packages from a repo or the AUR.",
		Run: func(args ...string) error {
			pacargs, pkgs, err := ParseSyncArgs(args...)
			if err != nil {
				return err
			}

			return InstallPkgs(pacargs, pkgs)
		},
	})

	RegisterCmd("-Si", &Cmd{
		Help: "Get info about a remote package.",
		Run: func(args ...string) error {
			pacargs, pkgs, err := ParseSyncArgs(args...)
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
				for _, arg := range args {
					if arg[0] != '-' {
						search = append(search, arg)
					}
				}

				info, err := AURSearch(strings.Join(search, " "))
				sc <- info
				errc <- err
			}()

			err := Pacman(append([]string{"-Ss"}, args...)...)
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

	runUpdate := func(arg string) func(...string) error {
		return func(args ...string) error {
			pacargs, pkgs, err := ParseSyncArgs(args...)
			if err != nil {
				return err
			}

			ac := make(chan []Pkg)
			errc := make(chan error)
			if pkgs == nil {
				go func() {
					pkgs, err := ListPackages()
					if err != nil {
						ac <- nil
						errc <- err
						return
					}

					var aurpkgs []Pkg
					for _, pkg := range pkgs {
						if info, ok := InAUR(pkg); ok {
							ver2, err := pkg.Version()
							if err != nil {
								ac <- nil
								errc <- err
								return
							}
							newer, err := Newer(info.GetInfo("Version"), ver2)
							if err != nil {
								ac <- nil
								errc <- err
								return
							}
							if newer {
								p, err := NewAURPkg(info)
								if err != nil {
									errc <- err
								}
								aurpkgs = append(aurpkgs, p)
							}
						}
					}

					ac <- aurpkgs
					errc <- nil
				}()
			}

			if pkgs == nil {
				err := SudoPacman(append([]string{arg}, pacargs...)...)
				if err != nil {
					<-ac
					<-errc
					return err
				}
			} else {
				err := InstallPkgs(args, pkgs)
				if err != nil {
					return err
				}
			}

			if pkgs == nil {
				aurpkgs := <-ac
				err = <-errc
				if err != nil {
					return err
				}

				for _, pkg := range aurpkgs {
					lp, err := NewLocalPkg(pkg.Name())
					if err != nil {
						return err
					}

					asdeps := ""
					if lp.IsDep() {
						asdeps = "--asdeps"
					}

					err := pkg.Install(nil, asdeps)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	RegisterCmd("-Su", &Cmd{
		Help: "Install updates.",
		Run:  runUpdate("-Su"),
	})

	RegisterCmd("-Syu", &Cmd{
		Help: "Update local package cache and install updates.",
		Run:  runUpdate("-Syu"),
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
