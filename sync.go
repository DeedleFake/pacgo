package main

func init() {
	RegisterCmd("-S", &Cmd{
		Help: "Install packages from a repo or the AUR.",
		Run: func(args ...string) error {
			var pacargs []string
			var pkgs []Pkg
			for _, arg := range args {
				if arg[0] != '-' {
					pkgs = append(pkgs, NewPkg(arg))
				} else {
					pacargs = append(pacargs, arg)
				}
			}

			return InstallPkgs(pacargs, pkgs)
		},
	})

	RegisterCmd("-Si", &Cmd{
		Help: "Get info about a remote package.",
		Run: func(args ...string) error {
			var pacargs []string
			var pkgs []Pkg
			for _, arg := range args {
				if arg[0] != '-' {
					pkgs = append(pkgs, NewPkg(arg))
				} else {
					pacargs = append(pacargs, arg)
				}
			}

			return InfoPkgs(pacargs, pkgs)
		},
	})
}
