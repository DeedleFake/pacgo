package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// A regular expression used to get version information out of the
	// outputs of the pacman -Qi and pacman -Si commands.
	VersionRE = regexp.MustCompile(`Version\s+:\s+(.*)`)
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

// Pkg represents a pacman package. This doesn't necessarily have to
// be a local package, or even a real package.
type Pkg interface {
	// Name returns the name of the package.
	Name() string

	// Version returns the full version string of the package and nil,
	// or "" and an error, if any.
	Version() (string, error)
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

func (err PkgNotFoundError) Error() string {
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
		return &PacmanPkg{
			name: name,
		}, nil
	}
	if info, ok := InAUR(name); ok {
		return NewAURPkg(info)
	}

	return nil, PkgNotFoundError{name}
}

// InLocal returns true if the named package is installed.
func InLocal(name string) bool {
	err := SilentPacman("-Qi", name)
	if err != nil {
		return false
	}

	return true
}

// InPacman returns true if the named package was found in the sync
// database.
func InPacman(name string) bool {
	err := SilentPacman("-Si", name)
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

// InstallPkgs installs the given pkgs using the given args. It
// returns an error, if any.
func InstallPkgs(args []string, pkgs []Pkg) error {
	var pacpkgs []string
	var other []InstallPkg
	for _, pkg := range pkgs {
		switch p := pkg.(type) {
		case *PacmanPkg:
			pacpkgs = append(pacpkgs, p.Name())
		case InstallPkg:
			other = append(other, p)
		default:
			return fmt.Errorf("Don't know how to install %v.", pkg.Name())
		}
	}

	if pacpkgs != nil {
		err := SudoPacman(append([]string{"-S"}, append(args, pacpkgs...)...)...)
		if err != nil {
			return err
		}
	}

	for _, pkg := range other {
		err := pkg.Install(nil, args...)
		if err != nil {
			return err
		}
	}

	return nil
}

// InfoPkgs prints the info for the given pkgs, using the given args.
// It returns an error, if any.
func InfoPkgs(args []string, pkgs []Pkg) error {
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

// AURPkg represents a package in the AUR.
type AURPkg struct {
	info     RPCResult
	pkgbuild *Pkgbuild
}

// NewAURPkg returns a *AURPkg using the given info. It returns an
// the *AURPkg and nil, or nil and an error, if any.
func NewAURPkg(info RPCResult) (*AURPkg, error) {
	rsp, err := http.Get(fmt.Sprintf(PKGURLFmt, info.GetInfo("Name")+"/PKGBUILD"))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	pb, err := ParsePkgbuild(rsp.Body)
	if err != nil {
		return nil, err
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

func (p *AURPkg) Install(dep Pkg, args ...string) (err error) {
	tmp, direrr := MkTmpDir(p.Name())

	cachefile := filepath.Join(tmp,
		p.Name(),
		fmt.Sprintf("%v-%v-%v.pkg.tar.xz",
			p.Name(),
			p.info.GetInfo("Version"),
			p.pkgbuild.LocalArch(),
		),
	)
	if _, err := os.Stat(cachefile); err != nil {
		cachefile = ""
	} else {
		var answer bool
		if dep == nil {
			answer, err = Caskf(true, "[c6]", "[c6]Found cached package for [c5]%v[c6]. Install?[ce]", p.Name())
			if err != nil {
				return err
			}
		} else {
			answer, err = Caskf(true, "[c6]", "[c6]Found cached package for [c5]%v[c6]. Install as dependency for [c5]%v[c6]?[ce]", p.Name(), dep.Name())
			if err != nil {
				return err
			}
		}
		if !answer {
			cachefile = ""
		}
	}

	if cachefile == "" {
		if direrr != nil {
			return direrr
		}

		if p.pkgbuild.HasDeps() {
			for _, dep := range append(p.pkgbuild.Deps, p.pkgbuild.MakeDeps...) {
				if !InLocal(dep) {
					pkg, err := NewRemotePkg(dep)
					if err != nil {
						if pnfe, ok := err.(PkgNotFoundError); ok {
							Cprintf("[c6]warning:[ce] Could not find package %v. Ignoring...\n", pnfe.PkgName)
						} else {
							return err
						}
					}
					if ap, ok := pkg.(*AURPkg); ok {
						err := ap.Install(p, "--asdeps")
						if err != nil {
							return err
						}
					}
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
	}

	err = MakepkgIn(filepath.Join(tmp, p.Name()), "-s", "-c", "-i")
	if err != nil {
		return err
	}

	os.RemoveAll(tmp)

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

// LocalPkg represents an installed package.
type LocalPkg struct {
	name string
}

// NewLocalPkg returns a *LocalPkg representing the named package and
// nil, or nil and an error, if any.
func NewLocalPkg(name string) (*LocalPkg, error) {
	if !InLocal(name) {
		return nil, errors.New(name + " is not installed.")
	}

	return &LocalPkg{
		name: name,
	}, nil
}

// ListLocalPkgs returns a slice containing all installed foreign
// packages and nil, or nil and an error, if any.
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

func (p *PkgbuildPkg) Install(dep Pkg, args ...string) error {
	if dep != nil {
		panic("How did that happen?")
	}

	for _, dep := range append(p.pkgbuild.Deps, p.pkgbuild.MakeDeps...) {
		if !InLocal(dep) {
			pkg, err := NewRemotePkg(dep)
			if err != nil {
				return err
			}
			if ap, ok := pkg.(*AURPkg); ok {
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
