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
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	// A regular expression used to get version information out of the
	// outputs of the pacman -Qi and pacman -Si commands.
	VersionRE = regexp.MustCompile(`Version\s+:\s+(.*)`)

	// A regular expression used to get the dependency list from the
	// outputs of the pacman -Qi and pacman -Si commands.
	DepsRE = regexp.MustCompile(`Depends\sOn\s\+:\s+(.*)`)
)

// Newer returns true, nil if ver1 is greater than ver2, else it
// returns false, nil. If any errors occur it returns false and an
// error.
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

// SamePkg returns true if the packages are the same.
func SamePkg(p1, p2 Pkg) bool {
	if !reflect.TypeOf(p1).AssignableTo(reflect.TypeOf(p2)) {
		return false
	}

	if p1.Name() != p2.Name() {
		return false
	}

	v1, _ := p1.Version()
	v2, _ := p2.Version()
	if v1 != v2 {
		return false
	}

	return true
}

// cleanName strips version information from the name of a package.
func cleanName(name string) string {
	for i := range name {
		if name[i] == '>' {
			return name[:i]
		}
	}

	return name
}

// Pkg represents a pacman package. This doesn't necessarily have to
// be a local package, or even a real package.
type Pkg interface {
	// Name returns the name of the package.
	Name() string

	// Version returns the full version string of the package and nil,
	// or "" and an error, if any.
	Version() (string, error)

	// Deps returns a list of the package's dependencies. It simply
	// ignores packages that it can't find. For this reason, it may
	// return a non-nil slice with a length of zero.
	//
	// Note that for installable packages the results of this method
	// are the packages that need to be installed before the package
	// can be installed. For example, for a *AURPkg, this is a
	// combination of both depends and makedeps.
	Deps() PkgList
}

// InstallPkg represents a Pkg that can be installed.
type InstallPkg interface {
	Pkg

	// Install installs the given package. The first argument is the
	// Pkg that this package is a dependency of. If it's nil then the
	// package is assumed to not be a dependency. The rest of the
	// arguments may differ depending on the implementation. It
	// returns an error, if any.
	Install(Pkg, ...string) error
}

// InfoPkg represents a Pkg capable of printing information about
// itself.
type InfoPkg interface {
	Pkg

	// Info prints the packages info. It returns an error, if any.
	Info(...string) error
}

// PkgNotFoundError is returned when functions can't find a certain
// package.
type PkgNotFoundError struct {
	PkgName string
}

func (err *PkgNotFoundError) Error() string {
	return "Package not found: " + err.PkgName
}

// NewRemotePkg returns a new Pkg representing the named package. It
// checks the local sync database first, and returns a *PacmanPkg and
// nil if it finds anything. If it doesn't, it tries the AUR. If it
// finds the package, it returns a *AURPkg and nil. Otherwise it
// returns nil and an error. If it is unable to find the package, it
// returns nil and a PkgNotFoundError.
func NewRemotePkg(name string) (Pkg, error) {
	if InPacman(name) {
		return NewPacmanPkg(name)
	}
	if info, ok := InAUR(name); ok {
		return NewAURPkg(info)
	}

	return nil, &PkgNotFoundError{name}
}

// InLocal returns true if the named package is installed.
func InLocal(name string) bool {
	err := SilentPacman("-Q", "--", name)
	if err != nil {
		return false
	}

	return true
}

// InPacman returns true if the named package was found in the sync
// database.
func InPacman(name string) bool {
	err := SilentPacman("-Si", "--", name)
	if err != nil {
		return false
	}

	return true
}

// InAUR checks for the named package in the AUR. If it finds it, it
// returns the RPCResult for its query and true, else if return an
// unspecified RPCResult and false.
func InAUR(name string) (RPCResult, bool) {
	info, err := AURInfo(name)
	if err != nil {
		return info, false
	}

	return info, true
}

// Provides returns a list of packages that provide the package
// specified or nil if none are found.
//
// TODO: This doesn't work at all. It needs to be completely redone.
//func Provides(pkg string) PkgList {
//	out, err := PacmanOutput("-Ssq", "^"+pkg+"$")
//	if err != nil {
//		return nil
//	}
//	out = bytes.TrimSpace(out)
//
//	lines := bytes.Split(out, []byte{'\n'})
//
//	pkgs := make(PkgList, 0, len(lines))
//	for _, line := range lines {
//		name := string(line)
//		if InPacman(name) {
//			pkg, err := NewPacmanPkg(name)
//			if err == nil {
//				pkgs = append(pkgs, pkg)
//			}
//		}
//	}
//
//	if len(pkgs) == 0 {
//		return nil
//	}
//
//	return pkgs
//}

// IsDep checks if the named package is installed as a dependency. It
// returns the result and nil, or false and an error, if any.
func IsDep(name string) (bool, error) {
	info, err := PacmanOutput("-Qi", name)
	if err != nil {
		return false, err
	}

	dep := bytes.Contains(info, []byte("Installed as a dependency"))

	return dep, nil
}

// Update checks for updates to the given Pkg. It returns a Pkg
// representing the new version and nil, or nil and an error, if any.
//
// TODO: Find a way to implement this as its own function that doesn't
//       needlessly slow everything down.
//func Update(pkg Pkg) (Pkg, error) {
//	switch p := pkg.(type) {
//	case *PacmanPkg:
//		return nil, errors.New("Can't check for updates to *PacmanPkg.")
//	case *AURPkg:
//		if (UpdateVCS) && (p.pkgbuild.IsVCS()) {
//			return pkg, nil
//		}
//		return nil, nil
//	case *LocalPkg:
//		r, err := NewRemotePkg(p.Name())
//		if err != nil {
//			return nil, err
//		}
//		rver, err := r.Version()
//		if err != nil {
//			return nil, err
//		}
//		pver, err := p.Version()
//		if err != nil {
//			return nil, err
//		}
//		up, err := Newer(rver, pver)
//		if err != nil {
//			return nil, err
//		}
//		if up {
//			return r, nil
//		}
//		return nil, nil
//	}
//
//	panic("Should never reach this point.")
//}

// InstallPkgs installs the given pkgs using the given args. It
// returns an error, if any.
func InstallPkgs(args []string, pkgs PkgList) error {
	var pacpkgs []string
	var other PkgList
	for _, pkg := range pkgs {
		switch p := pkg.(type) {
		case *PacmanPkg:
			pacpkgs = append(pacpkgs, p.Name())
		case InstallPkg:
			other = append(other, p)
		default:
			Cprintf("[c6]warning:[ce] Don't know how to install %v. Skipping.\n", pkg.Name())
		}
	}

	sortDone := make(chan bool)
	go func() {
		sort.Sort(other)
		sortDone <- true
	}()

	if pacpkgs != nil {
		err := AsRootPacman(append([]string{"-S"}, append(args, pacpkgs...)...)...)
		if err != nil {
			return err
		}
	}

	<-sortDone

	for _, pkg := range other {
		err := pkg.(InstallPkg).Install(nil, args...)
		if err != nil {
			Cprintf("[c6]warning:[ce] Installation of %v failed (%v). Skipping.\n", pkg.Name(), err)
		}
	}

	return nil
}

// InfoPkgs prints the info for the given pkgs, using the given args.
// It returns an error, if any.
func InfoPkgs(args []string, pkgs PkgList) error {
	for _, pkg := range pkgs {
		if ip, ok := pkg.(InfoPkg); ok {
			err := ip.Info(args...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// PacmanPkg represents a remote package in pacman's sync database.
type PacmanPkg struct {
	name string

	deps    PkgList
	gotDeps bool
}

// NewPacmanPkg returns a *PacmanPkg representing the named package
// and nil, or nil and an error, if any.
func NewPacmanPkg(name string) (*PacmanPkg, error) {
	return &PacmanPkg{
		name: name,
	}, nil
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
		return "", fmt.Errorf("PacmanPkg: Couldn't determine version of %v.", p.Name())
	}

	return string(bytes.TrimSpace(ver[1])), nil
}

func (p *PacmanPkg) Install(dep Pkg, args ...string) error {
	as := ""
	if dep != nil {
		as = "--asdeps"
	}

	err := AsRootPacman(append([]string{"-S", as}, args...)...)
	if err != nil {
		return err
	}

	return nil
}

func (p *PacmanPkg) Deps() (pl PkgList) {
	if p.gotDeps {
		return p.deps
	}

	lines, err := PacmanLines(true, "-Si", "--", p.Name())
	if err != nil {
		return nil
	}

	defer func() {
		if len(pl) == 0 {
			pl = nil
		}

		p.deps = pl
		p.gotDeps = true
	}()

	for _, line := range lines {
		match := DepsRE.FindSubmatch(line)
		if match != nil {
			names := bytes.Fields(match[1])

			pl = make(PkgList, 0, len(names))
			var pll sync.Mutex

			var wg sync.WaitGroup
			for _, name := range names {
				wg.Add(1)
				go func(name string) {
					defer wg.Done()

					pkg, err := NewRemotePkg(name)
					if err != nil {
						pkg, err := NewLocalPkg(name)
						if err != nil {
							return
						}

						pll.Lock()
						pl = append(pl, pkg)
						pll.Unlock()

						return
					}

					pll.Lock()
					pl = append(pl, pkg)
					pll.Unlock()
				}(string(name))
			}

			wg.Wait()

			return
		}
	}

	return nil
}

func (p *PacmanPkg) Info(args ...string) error {
	return Pacman(append([]string{"-Si"}, append(args, p.Name())...)...)
}

// AURPkg represents a package in the AUR.
type AURPkg struct {
	info     RPCResult
	pkgbuild *Pkgbuild

	deps    PkgList
	gotDeps bool
}

// NewAURPkg returns a *AURPkg using the given info. It returns an
// the *AURPkg and nil, or nil and an error, if any.
func NewAURPkg(info RPCResult) (*AURPkg, error) {
	rsp, err := http.Get(PKGURL(info.GetInfo("Name"), "PKGBUILD"))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	pb, err := ParsePkgbuild(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing %v's PKGBUILD: %v", info.GetInfo("Name"), err)
	}

	return &AURPkg{
		info:     info,
		pkgbuild: pb,
	}, nil
}

func (p *AURPkg) Name() string {
	return p.info.Results.(map[string]interface{})["Name"].(string)
}

func (p *AURPkg) Version() (string, error) {
	return p.info.GetInfo("Version"), nil
}

func (p *AURPkg) Deps() (pl PkgList) {
	if p.gotDeps {
		return p.deps
	}

	defer func() {
		if len(pl) == 0 {
			pl = nil
		}

		p.deps = pl
		p.gotDeps = true
	}()

	all := append(p.pkgbuild.Deps, p.pkgbuild.MakeDeps...)

	pl = make(PkgList, 0, len(all))
	var pll sync.Mutex

	var wg sync.WaitGroup
	for _, name := range all {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			pkg, err := NewRemotePkg(name)
			if err != nil {
				pkg, err := NewLocalPkg(name)
				if err != nil {
					return
				}

				pll.Lock()
				pl = append(pl, pkg)
				pll.Unlock()

				return
			}

			pll.Lock()
			pl = append(pl, pkg)
			pll.Unlock()
		}(name)
	}

	wg.Wait()

	return
}

func (p *AURPkg) Install(dep Pkg, args ...string) (err error) {
	tmp, direrr := MkTmpDir(p.Name())

	var isdep bool
	for _, arg := range args {
		if arg == "--asdeps" {
			isdep = true
			break
		}
	}

	genpkg := filepath.Join(tmp,
		p.Name(),
		fmt.Sprintf("%v-%v-%v.pkg.tar.xz",
			p.Name(),
			p.info.GetInfo("Version"),
			p.pkgbuild.LocalArch(),
		),
	)

	var cached bool
	if _, err := os.Stat(genpkg); !os.IsNotExist(err) {
		if dep == nil {
			cached, err = Caskf(true, "[c6]", ":: [c6]Found cached package for [c5]%v[c6]. Install?[ce]", p.Name())
			if err != nil {
				return err
			}
		} else {
			cached, err = Caskf(true, "[c6]", ":: [c6]Found cached package for [c5]%v[c6]. Install as dependency for [c5]%v[c6]?[ce]", p.Name(), dep.Name())
			if err != nil {
				return err
			}
		}
	}

	if !cached {
		if direrr != nil {
			return direrr
		}

		var answer bool
		if dep == nil {
			answer, err = Caskf(true, "[c6]", ":: [c6]Install [c5]%v[c6]?[ce]", p.Name())
			if err != nil {
				return
			}
		} else {
			answer, err = Caskf(true, "[c6]", ":: [c6]Install [c5]%v [c6]as a dependency for [c5]%v[c6]?[ce]",
				p.Name(),
				dep.Name(),
			)
			if err != nil {
				return
			}
		}
		if !answer {
			Cprintf("[c7]Skipping [c5]%v[c7]...[ce]\n\n", p.Name())
			return nil
		}

		Cprintf("[c2]==> [c1]Installing [c5]%v [c1]from the [c3]AUR[c1].[ce]\n", p.Name())

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
					pbpath := filepath.Join(tmp, p.Name(), "PKGBUILD")
					err := Edit(pbpath)
					if err != nil {
						return err
					}
					file, err := os.Open(pbpath)
					if err != nil {
						return fmt.Errorf("Unable to reload PKGBUILD for %v: %v", p.Name(), err)
					}
					p.pkgbuild, err = ParsePkgbuild(file)
					if err != nil {
						return fmt.Errorf("Unable to reload PKGBUILD for %v: %v", p.Name(), err)
					}
					genpkg = filepath.Join(tmp,
						p.Name(),
						fmt.Sprintf("%v-%v-%v.pkg.tar.xz",
							p.Name(),
							p.pkgbuild.VersionString(),
							p.pkgbuild.LocalArch(),
						),
					)
				} else {
					break
				}
			}

			if p.pkgbuild.HasInstall() {
				install := filepath.Join(tmp, p.Name(), p.pkgbuild.Install)
				if _, err := os.Stat(install); err == nil {
					for {
						answer, err := Caskf(false, "[c6]", ":: [c6]Edit [c5]%v [c6]using [c5]%v?[ce]",
							p.pkgbuild.Install,
							filepath.Base(EditPath),
						)
						if err != nil {
							return err
						}

						if answer {
							err := Edit(install)
							if err != nil {
								return err
							}
						} else {
							break
						}
					}
				} else {
					Cprintf("[c6]warning:[ce] Can't find %v install script.\n", install)
				}
			}
		}

		if p.pkgbuild.HasDeps() {
			deps := p.Deps()
			sort.Sort(deps)

			for _, dep := range deps {
				if ap, ok := dep.(*AURPkg); ok {
					if !InLocal(ap.Name()) {
						err := ap.Install(p, "--asdeps")
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if (dep == nil) && (!isdep) {
		err = MakepkgIn(filepath.Join(tmp, p.Name()), "-s", "-c", "-i")
		if err != nil {
			return err
		}
	} else {
		if cached {
			err = AsRootPacman("-U", "--asdeps", genpkg)
			if err != nil {
				return err
			}
		} else {
			err = MakepkgIn(filepath.Join(tmp, p.Name()), "-s", "-c")
			if err != nil {
				return err
			}

			file, err := os.Open(filepath.Join(tmp, p.Name(), "PKGBUILD"))
			if err != nil {
				return fmt.Errorf("Unable to reload PKGBUILD for %v: %v", p.Name(), err)
			}
			p.pkgbuild, err = ParsePkgbuild(file)
			if err != nil {
				return fmt.Errorf("Unable to reload PKGBUILD for %v: %v", p.Name(), err)
			}

			genpkg = filepath.Join(tmp,
				p.Name(),
				fmt.Sprintf("%v-%v-%v.pkg.tar.xz",
					p.Name(),
					p.pkgbuild.VersionString(),
					p.pkgbuild.LocalArch(),
				),
			)

			err = AsRootPacman("-U", "--asdeps", genpkg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *AURPkg) Info(args ...string) error {
	installscript := "No"
	if p.pkgbuild.HasInstall() {
		installscript = "Yes"
	}

	Cprintf("[c1]Repository     : [c3]aur[ce]\n")
	Cprintf("[c1]Name           : %v[ce]\n", p.info.GetInfo("Name"))
	Cprintf("[c1]Version        : [c2]%v[ce]\n", p.info.GetInfo("Version"))
	Cprintf("[c1]URL            : [c4]%v[ce]\n", p.info.GetInfo("URL"))
	Cprintf("[c1]Licenses       :[ce] %v\n", p.info.GetInfo("License"))
	Cprintf("[c1]Groups         :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Groups, " ")),
	)
	Cprintf("[c1]Provides       :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Provides, " ")),
	)
	Cprintf("[c1]Depends On     :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Deps, " ")),
	)
	Cprintf("[c1]Make Depends   :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.MakeDeps, " ")),
	)
	Cprintf("[c1]Check Depends  :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.CheckDeps, " ")),
	)
	Cprintf("[c1]Optional Deps  :[ce] %v\n",
		strings.Join(p.pkgbuild.OptDeps, "\n                 "),
	)
	Cprintf("[c1]Conflicts With :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Conflicts, " ")),
	)
	Cprintf("[c1]Replaces       :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Replaces, " ")),
	)
	Cprintf("[c1]Architecture   :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Arch, " ")),
	)
	Cprintf("[c1]Install Script :[ce] %v\n", installscript)
	Cprintf("[c1]Description    :[ce] %v\n", p.info.GetInfo("Description"))
	fmt.Println()

	return nil
}

func (p *AURPkg) IsVCS() bool {
	return p.pkgbuild.IsVCS()
}

// LocalPkg represents an installed package.
type LocalPkg struct {
	name string

	deps    PkgList
	gotDeps bool
}

// NewLocalPkg returns a *LocalPkg representing the named package and
// nil, or nil and an error, if any.
func NewLocalPkg(name string) (*LocalPkg, error) {
	if !InLocal(name) {
		return nil, errors.New(name + " is not installed. Can't make *LocalPkg.")
	}

	return &LocalPkg{
		name: name,
	}, nil
}

// ListForeignPkgs returns either a slice containing the names of all
// installed foreign packages and nil, or nil and an error, if any.
func ListForeignPkgs() ([]string, error) {
	lines, err := PacmanLines(true, "-Qqm")
	if err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, nil
	}

	list := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) != 0 {
			list = append(list, string(line))
		}
	}

	return list, nil
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
		return "", fmt.Errorf("LocalPkg: Couldn't determine version of %v.", p.Name())
	}

	return string(bytes.TrimSpace(ver[1])), nil
}

func (p *LocalPkg) Deps() (pl PkgList) {
	if p.gotDeps {
		return p.deps
	}

	lines, err := PacmanLines(true, "-Qi", "--", p.Name())
	if err != nil {
		return nil
	}

	defer func() {
		if len(pl) == 0 {
			pl = nil
		}

		p.deps = pl
		p.gotDeps = true
	}()

	for _, line := range lines {
		match := DepsRE.FindSubmatch(line)
		if match != nil {
			names := bytes.Fields(match[1])

			pl = make(PkgList, 0, len(names))
			var pll sync.Mutex

			var wg sync.WaitGroup
			for _, name := range names {
				wg.Add(1)
				go func(name string) {
					defer wg.Done()

					pkg, err := NewRemotePkg(name)
					if err != nil {
						pkg, err := NewLocalPkg(name)
						if err != nil {
							return
						}

						pll.Lock()
						pl = append(pl, pkg)
						pll.Unlock()

						return
					}

					pll.Lock()
					pl = append(pl, pkg)
					pll.Unlock()
				}(string(name))
			}

			wg.Wait()

			return
		}
	}

	return nil
}

func (p *LocalPkg) Info(args ...string) error {
	err := Pacman(append(append([]string{"-Qi"}, args...), p.Name())...)
	if err != nil {
		return err
	}

	return nil
}

// PkgbuildPkg represents a package that hasn't been built yet.
type PkgbuildPkg struct {
	pkgbuild *Pkgbuild

	deps    PkgList
	gotDeps bool
}

// NewPkgbuildPkg returns a *PkgbuildPkg representing the given
// PKGBUILD.
func NewPkgbuildPkg(pb *Pkgbuild) (*PkgbuildPkg, error) {
	return &PkgbuildPkg{
		pkgbuild: pb,
	}, nil
}

func (p *PkgbuildPkg) Name() string {
	return p.pkgbuild.Name
}

func (p *PkgbuildPkg) Version() (string, error) {
	epoch := ""
	if p.pkgbuild.Epoch != 0 {
		epoch = fmt.Sprintf("%v:", p.pkgbuild.Epoch)
	}

	return fmt.Sprintf("%v%v-%v", epoch, p.pkgbuild.Version, p.pkgbuild.Release), nil
}

func (p *PkgbuildPkg) Deps() (pl PkgList) {
	if p.gotDeps {
		return p.deps
	}

	defer func() {
		if len(pl) == 0 {
			pl = nil
		}

		p.deps = pl
		p.gotDeps = true
	}()

	all := append(p.pkgbuild.Deps, p.pkgbuild.MakeDeps...)

	pl = make(PkgList, 0, len(all))
	var pll sync.Mutex

	var wg sync.WaitGroup
	for _, name := range all {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			pkg, err := NewRemotePkg(name)
			if err != nil {
				pkg, err := NewLocalPkg(name)
				if err != nil {
					return
				}

				pll.Lock()
				pl = append(pl, pkg)
				pll.Unlock()

				return
			}

			pll.Lock()
			pl = append(pl, pkg)
			pll.Unlock()
		}(name)
	}

	wg.Wait()

	return
}

func (p *PkgbuildPkg) Install(dep Pkg, args ...string) error {
	if dep != nil {
		panic("How did that happen?")
	}

	depauth := false
argloop:
	for _, arg := range args {
		switch arg {
		case "-s", "--syncdeps":
			depauth = true
			break argloop
		}
	}

	// Just let makepkg fail if dependencies are missing.
	if depauth && p.pkgbuild.HasDeps() {
		deps := p.Deps()
		sort.Sort(deps)

		for _, dep := range deps {
			if ap, ok := dep.(*AURPkg); ok {
				err := ap.Install(p, "--asdeps")
				if err != nil {
					return err
				}
			}
		}
	}

	err := MakepkgIn("", args...)
	if err != nil {
		return err
	}

	return nil
}

func (p *PkgbuildPkg) Info(args ...string) error {
	ver, err := p.Version()
	if err != nil {
		return err
	}

	Cprintf("[c1]Name           : %v[ce]\n", p.pkgbuild.Name)
	Cprintf("[c1]Version        : [c2]%v[ce]\n", ver)
	Cprintf("[c1]URL            : [c4]%v[ce]\n", p.pkgbuild.URL)
	Cprintf("[c1]Licenses       :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Licenses, " ")),
	)
	Cprintf("[c1]Groups         :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Groups, " ")),
	)
	Cprintf("[c1]Provides       :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Provides, " ")),
	)
	Cprintf("[c1]Depends On     :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Deps, " ")),
	)
	Cprintf("[c1]Make Depends   :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.MakeDeps, " ")),
	)
	Cprintf("[c1]Check Depends  :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.CheckDeps, " ")),
	)
	Cprintf("[c1]Optional Deps  :[ce] %v\n",
		strings.Join(p.pkgbuild.OptDeps, "\n                 "),
	)
	Cprintf("[c1]Conflicts With :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Conflicts, " ")),
	)
	Cprintf("[c1]Replaces       :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Replaces, " ")),
	)
	Cprintf("[c1]Architecture   :[ce] %v\n",
		strings.TrimSpace(strings.Join(p.pkgbuild.Arch, " ")),
	)
	Cprintf("[c1]Install Script :[ce] %v\n", p.pkgbuild.InstallScript())
	Cprintf("[c1]Description    :[ce] %v\n", p.pkgbuild.Description)
	fmt.Println()

	return nil
}
