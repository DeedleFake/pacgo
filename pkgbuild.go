package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
)

const pkgbuildScan = `echo "name:$pkgname"
echo "ver:$pkgver"
echo "rel:$pkgrel"

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

type Pkgbuild struct {
	Name      string
	Version   string
	Release   int
	Deps      []string
	MakeDeps  []string
	OptDeps   []string
	Conflicts []string
	Replaces  []string
	Arch      []string
}

func ParsePkgbuild(r io.Reader) (*Pkgbuild, error) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		return nil, errors.New("Couldn't find bash.")
	}

	cmd := &exec.Cmd{
		Path: bash,
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

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	_, err = inpipe.Write(buf)
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

	return pb, nil
}
