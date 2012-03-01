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
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

// A bash script that echos parts of a PKGBUILD in a more parsable
// format.
const pkgbuildScan = `echo "name:$pkgname"
echo "ver:$pkgver"
echo "rel:$pkgrel"
echo "epoch:$epoch"
echo "url:$url"

echo "licenselen:${#license[*]}"
for ((i=0; i<${#license}; i++)); do
	echo "license:${license[i]}"
done

echo "grouplen:${#groups[*]}"
for ((i=0; i<${#groups[*]}; i++)); do
	echo "group:${groups[i]}"
done

echo "providelen:${#provides[*]}"
for ((i=0; i<${#provides[*]}; i++)); do
	echo "provide:${provides[i]}"
done

echo "deplen:${#depends[*]}"
for ((i=0; i<${#depends[*]}; i++)); do
	echo "dep:${depends[i]}"
done

echo "makedeplen:${#makedepends[*]}"
for ((i=0; i<${#makedepends[*]}; i++)); do
	echo "makedep:${makedepends[i]}"
done

echo "checkdeplen:${#checkdepends[*]}"
for ((i=0; i<${#checkdepends[*]}; i++)); do
	echo "checkdep:${checkdepends[i]}"
done

echo "optdeplen:${#optdeplen[*]}"
for ((i=0; i<${#optdepends[*]}; i++)); do
	echo "optdep:${optdepends[i]}"
done

echo "conflictlen:${#conflicts[*]}"
for ((i=0; i<${#conflicts}; i++)); do
	echo "conflict:${conflicts[i]}"
done

echo "repllen:${#replaces[*]}"
for ((i=0; i<${#replaces}; i++)); do
	echo "repl:${replaces[i]}"
done

echo "archlen:${#arch[*]}"
for ((i=0; i<${#arch}; i++)); do
	echo "arch:${arch[i]}"
done

echo "install:$install"
echo "desc:$pkgdesc"

if [[ -n "$_darcstrunk" && -n "$_darcsmod" ]]; then
	echo "vcs:darcs"
elif [[ -n "$_cvsroot" && -n "$_cvsmod" ]]; then
	echo "vcs:cvs"
elif [[ -n "$_gitroot" && -n "$_gitname" ]]; then
	echo "vcs:git"
elif [[ -n "$_svntrunk" && -n "$_svnmod" ]]; then
	echo "vcs:svn"
elif [[ -n "$_bzrtrunk" && -n "$_bzrmod" ]]; then
	echo "vcs:bzr"
elif [[ -n "$_hgroot" && -n "$_hgrepo" ]]; then
	echo "vcs:hg"
fi

exit
`

// Pkgbuild represents a PKGBUILD.
type Pkgbuild struct {
	Name        string
	Version     string
	Release     int
	Epoch       int
	URL         string
	Licenses    []string
	Groups      []string
	Provides    []string
	Deps        []string
	MakeDeps    []string
	CheckDeps   []string
	OptDeps     []string
	Conflicts   []string
	Replaces    []string
	Arch        []string
	Install     string
	Description string
	VCS         string
}

// ParsePkgbuild parses a PKGBUILD read from r. It returns a *Pkgbuild
// and nil, or nil and an error, if any.
func ParsePkgbuild(r io.Reader) (*Pkgbuild, error) {
	cmd := &exec.Cmd{
		Path: BashPath,
		Args: []string{filepath.Base(BashPath), "--login"},
	}

	inpipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	outpipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(inpipe, "source /etc/makepkg.conf\n")
	if err != nil {
		return nil, err
	}

	u, err := user.Current()
	if err == nil {
		hmc := filepath.Join(u.HomeDir, ".makepkg.conf")
		if _, err := os.Stat(hmc); err == nil {
			_, err = io.WriteString(inpipe, "source '" + hmc + "'\n")
			if err != nil {
				return nil, err
			}
		}
	}

	_, err = io.Copy(inpipe, r)
	if err != nil {
		return nil, err
	}

	// Work around PKGBUILDs that don't end with a newline.
	_, err = inpipe.Write([]byte{'\n'})
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(inpipe, pkgbuildScan)
	if err != nil {
		return nil, err
	}

	lines, err := ReadLines(outpipe, true)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	pb := new(Pkgbuild)

	for _, line := range lines {
		parts := bytes.SplitN(line, []byte{':'}, 2)
		switch string(parts[0]) {
		case "name":
			pb.Name = string(bytes.TrimSpace(parts[1]))
		case "ver":
			pb.Version = string(bytes.TrimSpace(parts[1]))
		case "rel":
			rel, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad $pkgrel: %v.", ne.Num)
			}
			pb.Release = int(rel)
		case "epoch":
			if str := string(bytes.TrimSpace(parts[1])); str == "" {
				pb.Epoch = 0
			} else {
				epoch, err := strconv.ParseInt(str, 10, 0)
				if err != nil {
					ne := err.(*strconv.NumError)
					return nil, fmt.Errorf("Got bad $epoch: %v.", ne.Num)
				}
				pb.Epoch = int(epoch)
			}
		case "url":
			pb.URL = string(bytes.TrimSpace(parts[1]))
		case "licenselen":
			licenselen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad licenselen: %v.", ne.Num)
			}
			pb.Licenses = make([]string, licenselen)
		case "license":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Licenses = append(pb.Licenses, str)
			}
		case "grouplen":
			grouplen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad grouplen: %v.", ne.Num)
			}
			pb.Groups = make([]string, grouplen)
		case "group":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Groups = append(pb.Groups, str)
			}
		case "providelen":
			providelen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad providelen: %v.", ne.Num)
			}
			pb.Provides = make([]string, providelen)
		case "provide":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Provides = append(pb.Provides, str)
			}
		case "deplen":
			deplen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad deplen: %v.", ne.Num)
			}
			pb.Deps = make([]string, 0, deplen)
		case "dep":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Deps = append(pb.Deps, str)
			}
		case "makedeplen":
			makedeplen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad makedeplen: %v.", ne.Num)
			}
			pb.MakeDeps = make([]string, 0, makedeplen)
		case "makedep":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.MakeDeps = append(pb.MakeDeps, str)
			}
		case "checkdeplen":
			checkdeplen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad checkdeplen: %v.", ne.Num)
			}
			pb.CheckDeps = make([]string, 0, checkdeplen)
		case "checkdep":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.CheckDeps = append(pb.CheckDeps, str)
			}
		case "optdeplen":
			optdeplen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad optdeplen: %v.", ne.Num)
			}
			pb.OptDeps = make([]string, 0, optdeplen)
		case "optdep":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.OptDeps = append(pb.OptDeps, str)
			}
		case "conflictlen":
			conflictlen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad conflictlen: %v.", ne.Num)
			}
			pb.Conflicts = make([]string, 0, conflictlen)
		case "conflict":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Conflicts = append(pb.Conflicts, str)
			}
		case "repllen":
			repllen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad repllen: %v.", ne.Num)
			}
			pb.Replaces = make([]string, 0, repllen)
		case "repl":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Replaces = append(pb.Replaces, str)
			}
		case "archlen":
			archlen, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				ne := err.(*strconv.NumError)
				return nil, fmt.Errorf("Got bad archlen: %v.", ne.Num)
			}
			pb.Arch = make([]string, 0, archlen)
		case "arch":
			if str := string(bytes.TrimSpace(parts[1])); str != "" {
				pb.Arch = append(pb.Arch, str)
			}
		case "install":
			pb.Install = string(bytes.TrimSpace(parts[1]))
		case "desc":
			pb.Description = string(bytes.TrimSpace(parts[1]))
		case "vcs":
			pb.VCS = string(bytes.TrimSpace(parts[1]))
		}
	}

	if len(pb.Licenses) == 0 {
		pb.Licenses = []string{"None"}
	}
	if len(pb.Groups) == 0 {
		pb.Groups = []string{"None"}
	}
	if len(pb.Provides) == 0 {
		pb.Provides = []string{"None"}
	}
	if len(pb.Deps) == 0 {
		pb.Deps = []string{"None"}
	}
	if len(pb.MakeDeps) == 0 {
		pb.MakeDeps = []string{"None"}
	}
	if len(pb.CheckDeps) == 0 {
		pb.CheckDeps = []string{"None"}
	}
	if len(pb.OptDeps) == 0 {
		pb.OptDeps = []string{"None"}
	}
	if len(pb.Conflicts) == 0 {
		pb.Conflicts = []string{"None"}
	}
	if len(pb.Replaces) == 0 {
		pb.Replaces = []string{"None"}
	}
	if len(pb.Arch) == 0 {
		return nil, errors.New("PKGBUILD doesn't have an arch.")
	}

	return pb, nil
}

// HasDeps returns true if the *Pkgbuild has any deps.
func (p *Pkgbuild) HasDeps() bool {
	if (len(p.Deps) == 1) && (p.Deps[0] == "None") && (len(p.MakeDeps) == 1) && (p.MakeDeps[0] == "None") {
		return false
	}

	return true
}

// HasInstall returns true if the *Pkgbuild specifies an install script.
func (p *Pkgbuild) HasInstall() bool {
	return p.Install != ""
}

// InstallScript is a convienece function that is used for printing
// PKGBUILD info. If p.Install is blank, it returns "None", otherwise
// it returns p.Install.
func (p *Pkgbuild) InstallScript() string {
	if p.Install == "" {
		return "None"
	}

	return p.Install
}

// LocalArch returns the arch string for the *Pkgbuild that a package built on the local machine using the PKGBUILD would be likely to have.
func (p *Pkgbuild) LocalArch() string {
	if (len(p.Arch) == 1) && (p.Arch[0] == "any") {
		return "any"
	}

	var find string
	switch runtime.GOARCH {
	case "386":
		find = "i686"
	case "amd64":
		find = "x86_64"
	default:
		return ""
	}

	for _, a := range p.Arch {
		if a == find {
			return find
		}
	}

	return ""
}

// IsVCS returns true if p represents a VCS PKGBUILD.
func (p *Pkgbuild) IsVCS() bool {
	return len(p.VCS) > 0
}
