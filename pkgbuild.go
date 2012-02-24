package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strconv"
)

// A bash script that echos parts of a PKGBUILD in a more parsable
// format.
const pkgbuildScan = `echo "name:$pkgname"
echo "ver:$pkgver"
echo "rel:$pkgrel"
echo "epoch:$epoch"

for ((i=0; i<${#depends[@]}; i++)); do
	echo "dep:${depends[i]}"
done

for ((i=0; i<${#makedepends[@]}; i++)); do
	echo "makedep:${makedepends[i]}"
done

for ((i=0; i<${#optdepends[@]}; i++)); do
	echo "optdep:${optdepends[i]}"
done

for ((i=0; i<${#conflicts}; i++)); do
	echo "conflict:${conflicts[i]}"
done

for ((i=0; i<${#replaces}; i++)); do
	echo "repl:${replaces[i]}"
done

for ((i=0; i<${#arch}; i++)); do
	echo "arch:${arch[i]}"
done

exit
`

// Pkgbuild represents a PKGBUILD.
type Pkgbuild struct {
	Name      string
	Version   string
	Release   int
	Epoch     int
	Deps      []string
	MakeDeps  []string
	OptDeps   []string
	Conflicts []string
	Replaces  []string
	Arch      []string

	//Raw []byte
}

// ParsePkgbuild parses a PKGBUILD read from r. It returns a *Pkgbuild
// and nil, or nil and an error, if any.
func ParsePkgbuild(r io.Reader) (*Pkgbuild, error) {
	cmd := &exec.Cmd{
		Path: BashPath,
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

	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	_, err = inpipe.Write(raw)
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(inpipe, pkgbuildScan)
	if err != nil {
		return nil, err
	}

	out, err := ioutil.ReadAll(outpipe)
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSpace(out)

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	pb := new(Pkgbuild)

	lines := bytes.Split(out, []byte{'\n'})
	for _, line := range lines {
		parts := bytes.SplitN(line, []byte{':'}, 2)
		switch string(parts[0]) {
		case "name":
			pb.Name = string(parts[1])
		case "ver":
			pb.Version = string(parts[1])
		case "rel":
			rel, err := strconv.ParseInt(string(parts[1]), 10, 0)
			if err != nil {
				return nil, errors.New("Bad $pkgrel in PKGBUILD.")
			}
			pb.Release = int(rel)
		case "epoch":
			if str := string(parts[1]); str == "" {
				pb.Epoch = 0
			} else {
				epoch, err := strconv.ParseInt(str, 10, 0)
				if err != nil {
					return nil, errors.New("Bad $epoch in PKGBUILD.")
				}
				pb.Epoch = int(epoch)
			}
		case "dep":
			pb.Deps = append(pb.Deps, string(parts[1]))
		case "makedep":
			pb.MakeDeps = append(pb.MakeDeps, string(parts[1]))
		case "optdep":
			pb.OptDeps = append(pb.OptDeps, string(parts[1]))
		case "conflict":
			pb.Conflicts = append(pb.Conflicts, string(parts[1]))
		case "repl":
			pb.Replaces = append(pb.Replaces, string(parts[1]))
		case "arch":
			pb.Arch = append(pb.Arch, string(parts[1]))
		}
	}

	if pb.Deps == nil {
		pb.Deps = []string{"None"}
	}
	if pb.MakeDeps == nil {
		pb.MakeDeps = []string{"None"}
	}
	if pb.OptDeps == nil {
		pb.OptDeps = []string{"None"}
	}
	if pb.Conflicts == nil {
		pb.Conflicts = []string{"None"}
	}
	if pb.Replaces == nil {
		pb.Replaces = []string{"None"}
	}
	if pb.Arch == nil {
		return nil, errors.New("PKGBUILD doesn't have an arch.")
	}

	//pb.Raw = raw

	return pb, nil
}

//func (p *Pkgbuild) WriteTo(w io.Writer) (int, error) {
//	return w.Write(p.Raw)
//}

// HasDeps returns true if the *Pkgbuild has any deps.
func (p *Pkgbuild) HasDeps() bool {
	if (len(p.Deps) == 1) && (p.Deps[0] == "None") {
		return false
	}

	return true
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
