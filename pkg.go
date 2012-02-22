package main

import (
	"errors"
	"fmt"
	"strings"
)

type Pkg interface {
	Name() string
	Install(...string) error
	Info(...string) error
}

func NewRemotePkg(name string) (Pkg, error) {
	if InPacman(name) {
		return &PacmanPkg{
			name: name,
		}, nil
	}
	if info, ok := InAUR(name); ok {
		return NewAURPkg(info)
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

func InAUR(name string) (RPCResult, bool) {
	info, err := AURInfo(name)
	if err != nil {
		return info, false
	}

	return info, true
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
	name string
}

func (p *PacmanPkg) Name() string {
	return p.name
}

func (p *PacmanPkg) Install(args ...string) error {
	return InstallPkgs(args, []Pkg{p})
}

func (p *PacmanPkg) Info(args ...string) error {
	return Pacman(append([]string{"-Si"}, append(args, p.Name())...)...)
}

type AURPkg struct {
	info     RPCResult
	pkgbuild *Pkgbuild
}

func (p *AURPkg) Name() string {
	return p.info.Results.(map[string]interface{})["Name"].(string)
}

func (p *AURPkg) Install(args ...string) error {
	return errors.New("Not implemented.")
}

func (p *AURPkg) Info(args ...string) error {
	Cprintf("[c1]Repository     : [c3]aur[ce]\n")
	Cprintf("[c1]Name           : %v[ce]\n", p.info.GetInfo("Name"))
	Cprintf("[c1]Version        : [c2]%v[ce]\n", p.info.GetInfo("Version"))
	Cprintf("[c1]URL            : [c4]%v[ce]\n", p.info.GetInfo("URL"))
	Cprintf("[c1]Licenses       :[ce] %v\n", p.info.GetInfo("License"))
	Cprintf("[c1]Depends On     :[ce] %v\n", strings.Join(p.pkgbuild.Deps, " "))
	fmt.Println()

	return nil
}
