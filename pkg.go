package main

type Pkg interface {
	InstallFunc() func(...string) error
	IsDep() bool
}

func NewPkg(name string, aurdep bool) Pkg {
	return &PacmanPkg{
		Name:    name,
		Dep:     aurdep,
		Depends: nil,
	}
}

func AsWhat(p Pkg) string {
	if p.IsDep() {
		return ""
	}

	return "--asdeps"
}

func InstallPkgs(args []string, pkgs []Pkg) error {
	var pkgargs []string
	for _, pkg := range pkgs {
		switch p := pkg.(type) {
		case *PacmanPkg:
			pkgargs = append(pkgargs, p.Name)
		}
	}

	err := SudoPacman(append([]string{"-S"}, append(args, pkgargs...)...)...)
	if err != nil {
		return err
	}

	return nil
}

type PacmanPkg struct {
	Name    string
	Dep     bool
	Depends []Pkg
}

func (p *PacmanPkg) InstallFunc() func(...string) error {
	return nil
}

func (p *PacmanPkg) IsDep() bool {
	return p.Dep
}
