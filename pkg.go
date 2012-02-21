package main

type Pkg interface {
	Name() string
	InstallFunc() func(...string) error
	Info(...string) error
}

func NewPkg(name string) Pkg {
	return &PacmanPkg{
		name:    name,
		Dep:     false,
		Depends: nil,
	}
}

func InstallPkgs(args []string, pkgs []Pkg) error {
	var pacpkgs []string
	for _, pkg := range pkgs {
		switch p := pkg.(type) {
		case *PacmanPkg:
			pacpkgs = append(pacpkgs, p.Name())
		}
	}

	err := SudoPacman(append([]string{"-S"}, append(args, pacpkgs...)...)...)
	if err != nil {
		return err
	}

	return nil
}

func InfoPkgs(args []string, pkgs []Pkg) error {
	for _, pkg := range pkgs {
		err := pkg.Info(args...)
		if err != nil {
			return err
		}
	}

	return nil
}

type PacmanPkg struct {
	name    string
	Dep     bool
	Depends []Pkg
}

func (p *PacmanPkg) Name() string {
	return p.name
}

func (p *PacmanPkg) InstallFunc() func(...string) error {
	return nil
}

func (p *PacmanPkg) Info(args ...string) error {
	return Pacman(append([]string{"-Si"}, append(args, p.Name())...)...)
}
