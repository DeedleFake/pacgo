package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	VersionRE = regexp.MustCompile(`Version\s+:\s+(.*)`)
)

func Newer(ver1, ver2 string) (bool, error) {
	out, err := VercmpOutput(ver1, ver2)
	if err != nil {
		return false, err
	}
	out = bytes.TrimSpace(out)

	switch string(out) {
	case "-1", "0":
		return false, nil
	case "1":
		return true, nil
	}

	panic("Bad vercmp output: " + string(out))
}

type Pkg interface {
	Name() string
	Version() (string, error)

	Install(Pkg, ...string) error
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

func Update(pkg Pkg) (Pkg, error) {
	switch p := pkg.(type) {
	case *PacmanPkg:
		return nil, errors.New("Unable to tell if update is available.")
	case *AURPkg:
		if (UpdateDevel) && (p.IsDevel()) {
			return pkg, nil
		}
		return nil, nil
	case *LocalPkg:
		r, err := NewRemotePkg(p.Name())
		if err != nil {
			return nil, err
		}
		rver, err := r.Version()
		if err != nil {
			return nil, err
		}
		pver, err := p.Version()
		if err != nil {
			return nil, err
		}
		up, err := Newer(rver, pver)
		if err != nil {
			return nil, err
		}
		if up {
			return r, nil
		}
		return nil, nil
	}

	panic("Should never reach this point.")
}

func InstallPkgs(args []string, pkgs []Pkg) error {
	var pacpkgs []string
	for _, pkg := range pkgs {
		switch p := pkg.(type) {
		case *PacmanPkg:
			pacpkgs = append(pacpkgs, p.Name())
		}
	}

	if pacpkgs != nil {
		err := SudoPacman(append([]string{"-S"}, append(args, pacpkgs...)...)...)
		if err != nil {
			return err
		}
	}

	for _, pkg := range pkgs {
		switch p := pkg.(type) {
		case *AURPkg:
			err := p.Install(nil, args...)
			if err != nil {
				return err
			}
		}
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

func (p *PacmanPkg) Version() (string, error) {
	info, err := PacmanOutput("-Si", p.Name())
	if err != nil {
		return "", err
	}

	ver := VersionRE.FindSubmatch(info)
	if ver == nil {
		return "", errors.New("Couldn't determine version.")
	}

	return string(bytes.TrimSpace(ver[1])), nil
}

func (p *PacmanPkg) Install(dep Pkg, args ...string) error {
	asdeps := ""
	if dep != nil {
		asdeps = "--asdeps"
	}

	err := SudoPacman(append([]string{"-S", asdeps}, args...)...)
	if err != nil {
		return err
	}

	return nil
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

func (p *AURPkg) Version() (string, error) {
	return p.info.GetInfo("Version"), nil
}

func (p *AURPkg) Install(dep Pkg, args ...string) (err error) {
	if p.pkgbuild.HasDeps() {
		var deps []Pkg
		for _, dep := range p.pkgbuild.Deps {
			if !InLocal(dep) {
				pkg, err := NewRemotePkg(dep)
				if err != nil {
					return err
				}
				if _, ok := pkg.(*AURPkg); ok {
					deps = append(deps, pkg)
				}
			}
		}
		for _, dep := range deps {
			err := dep.Install(p, "--asdeps")
			if err != nil {
				return err
			}
		}
	}

	var answer bool
	if dep == nil {
		answer, err = Caskf(true, "[c6]", "[c6]Install [c5]%v[c6]?[ce]", p.Name())
		if err != nil {
			return
		}
	} else {
		answer, err = Caskf(true, "[c6]", "[c6]Install [c5]%v [c6]as a dependency for [c5]%v[c6]?[ce]",
			p.Name(),
			dep.Name(),
		)
		if err != nil {
			return
		}
	}
	if !answer {
		return nil
	}

	Cprintf("[c2]==> [c1]Installing [c5]%v [c1]from the [c3]AUR[c1].[ce]\n", p.Name())

	tmp, err := MkTmpDir(p.Name())
	if err != nil {
		return errors.New("Failed to create temporary dir.")
	}
	defer os.RemoveAll(tmp)

	tr, err := GetSourceTar(p.Name())
	if err != nil {
		return err
	}

	err = ExtractTar(tmp, tr)
	if err != nil {
		return err
	}

	if EditPath != "" {
		for {
			answer, err := Caskf(false, "[c6]", "[c6]Edit [c5]PKGBUILD [c6]using [c5]%v?[ce]", filepath.Base(EditPath))
			if err != nil {
				return err
			}

			if answer {
				err := Edit(filepath.Join(tmp, p.Name(), "PKGBUILD"))
				if err != nil {
					return err
				}
			} else {
				break
			}
		}

		install := filepath.Join(tmp, p.Name(), p.Name()+".install")
		if _, err := os.Stat(install); err == nil {
			for {
				answer, err := Caskf(false, "[c6]Edit [c5]%v [c6]using [c5]%v?[ce]",
					filepath.Base(install),
					filepath.Base(EditPath),
				)
				if err != nil {
					return err
				}

				if answer {
					err := Edit(filepath.Join(tmp, p.Name(), install))
					if err != nil {
						return err
					}
				} else {
					break
				}
			}
		}
	}

	err = MakepkgIn(filepath.Join(tmp, p.Name()), "-s", "-c", "-i")
	if err != nil {
		return err
	}

	return nil
}

func (p *AURPkg) Info(args ...string) error {
	Cprintf("[c1]Repository     : [c3]aur[ce]\n")
	Cprintf("[c1]Name           : %v[ce]\n", p.info.GetInfo("Name"))
	Cprintf("[c1]Version        : [c2]%v[ce]\n", p.info.GetInfo("Version"))
	Cprintf("[c1]URL            : [c4]%v[ce]\n", p.info.GetInfo("URL"))
	Cprintf("[c1]Licenses       :[ce] %v\n", p.info.GetInfo("License"))
	Cprintf("[c1]Depends On     :[ce] %v\n", strings.Join(p.pkgbuild.Deps, " "))
	Cprintf("[c1]Make Depends   :[ce] %v\n", strings.Join(p.pkgbuild.MakeDeps, " "))
	Cprintf("[c1]Optional Deps  :[ce] %v\n",
		strings.Join(p.pkgbuild.OptDeps, "\n                 "),
	)
	Cprintf("[c1]Conflicts With :[ce] %v\n", strings.Join(p.pkgbuild.Conflicts, " "))
	Cprintf("[c1]Replaces       :[ce] %v\n", strings.Join(p.pkgbuild.Replaces, " "))
	Cprintf("[c1]Architecture   :[ce] %v\n", strings.Join(p.pkgbuild.Arch, " "))
	Cprintf("[c1]Description    :[ce] %v\n", p.info.GetInfo("Description"))
	fmt.Println()

	return nil
}

func (p *AURPkg) IsDevel() bool {
	panic("Not implemented.")
}

type LocalPkg struct {
	name string
}

func NewLocalPkg(name string) (*LocalPkg, error) {
	if !InLocal(name) {
		return nil, errors.New(name + " is not installed.")
	}

	return &LocalPkg{
		name: name,
	}, nil
}

func ListLocalPkgs() ([]*LocalPkg, error) {
	list, err := PacmanOutput("-Qqm")
	if err != nil {
		return nil, err
	}
	list = bytes.TrimSpace(list)

	var pkgs []*LocalPkg
	lines := bytes.Split(list, []byte("\n"))
	for _, line := range lines {
		pkg, err := NewLocalPkg(string(line))
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

func (p *LocalPkg) Name() string {
	return p.name
}

func (p *LocalPkg) Version() (string, error) {
	info, err := PacmanOutput("-Qi", p.Name())
	if err != nil {
		return "", err
	}

	ver := VersionRE.FindSubmatch(info)
	if ver == nil {
		return "", errors.New("Couldn't determine version.")
	}

	return string(bytes.TrimSpace(ver[1])), nil
}

func (p *LocalPkg) Install(Pkg, ...string) error {
	panic("Not implemented.")
}

func (p *LocalPkg) Info(args ...string) error {
	err := Pacman(append(append([]string{"-Qi"}, args...), p.Name())...)
	if err != nil {
		return err
	}

	return nil
}

func (p *LocalPkg) IsDep() (bool, error) {
	info, err := PacmanOutput("-Qi", p.Name())
	if err != nil {
		return false, err
	}

	dep := bytes.Contains(info, []byte("Installed as a dependency"))

	return dep, nil
}
