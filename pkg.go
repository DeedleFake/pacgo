package main

import (
	"errors"
	"fmt"
	"strings"
)

type InstallFunc func(...string) error

type Pkg interface {
	Name() string
	InstallFunc() InstallFunc
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

func (p *PacmanPkg) InstallFunc() InstallFunc {
	return func(args ...string) error {
		return InstallPkgs(args, []Pkg{p})
	}
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

func (p *AURPkg) InstallFunc() InstallFunc {
	return nil
}

func (p *AURPkg) Info(args ...string) error {
	fmt.Printf("Repository     : aur\n")
	fmt.Printf("Name           : %v\n", p.info.Get("Name"))
	fmt.Printf("Version        : %v\n", p.info.Get("Version"))
	fmt.Printf("URL            : %v\n", p.info.Get("URL"))
	fmt.Printf("Licenses       : %v\n", p.info.Get("License"))
	fmt.Printf("Depends On     : %v\n", strings.Join(p.pkgbuild.Deps, " "))
	fmt.Println()

	return nil
}
