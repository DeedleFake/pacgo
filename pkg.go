package main

import (
	"errors"
)

type InstallFunc func(...string) error

type Pkg interface {
	Name() string
	InstallFunc() InstallFunc
	Info(...string) error
}

func NewPkg(name string) (Pkg, error) {
	if InPacman(name) {
		return &PacmanPkg{
			name:    name,
			Dep:     false,
			Depends: nil,
		}, nil
	}
	if InAUR(name) {
		return &AURPkg{
			name:    name,
			Dep:     false,
			Depends: nil,
		}, nil
	}

	return nil, errors.New("No such package: " + name)
}

func InLocal(name string) bool {
	err := SilentPacman("-Qi", name)
	if err != nil {
		return false
	}

	return true
}

func InPacman(name string) bool {
	err := SilentPacman("-Si", name)
	if err != nil {
		return false
	}

	return true
}

func InAUR(name string) bool {
	return false
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

func (p *PacmanPkg) InstallFunc() InstallFunc {
	return nil
}

func (p *PacmanPkg) Info(args ...string) error {
	return Pacman(append([]string{"-Si"}, append(args, p.Name())...)...)
}

type AURPkg struct {
	name    string
	Dep     bool
	Depends []Pkg
}

func (p *AURPkg) Name() string {
	return p.name
}

func (p *AURPkg) InstallFunc() InstallFunc {
	return nil
}

func (p *AURPkg) Info(args ...string) error {
	return errors.New("Not implemented.")
}
