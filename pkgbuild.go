package main

import (
	"errors"
	"io"
	//"io/ioutil"
	"regexp"
)

var (
	PkgbuildNameRE = regexp.MustCompile(`pkgname=([a-zA-Z0-9]+)`)
	PkgbuildDepsRE = regexp.MustCompile(`(make|opt)?depends=\((['"]?([a-zA-Z0-9=<>]+)['"]?\n?)*\)`)
)

type Pkgbuild struct {
	Name     string
	Version  string
	Release  int
	Deps     []string
	MakeDeps []string
	OptDeps  []string
}

func ParsePkgbuild(r io.Reader) (*Pkgbuild, error) {
	return nil, errors.New("Not implemented.")

	//buf, err := ioutil.ReadAll(r)
	//if err != nil {
	//	return nil, err
	//}

	//name := PkgbuildNameRE.FindAllSubmatch(buf, -1)
	//if name == nil {
	//	return nil, errors.New("Couldn't get pkgname from PKGBUILD.")
	//}

	//deps := PkgbuildDepsRE.FindAllSubmatch(buf, -1)
	//if deps == nil {
	//	none := [][]byte{nil, []byte("None")}
	//	deps = [][][]byte{
	//		none,
	//		append([]byte("make"), none...),
	//		append([]byte("opt"), none...),
	//	}
	//}

	//pb := &Pkgbuild{
	//	Name: string(name[0][0]),
	//}

	//pb.Deps = make([]string, 0, len(deps))
	//for _, set := range deps {
	//	switch string(set[0]) {
	//	case "make":
	//	case "opt":
	//	default:
	//		for _, dep := range deps[1:] {
	//			pb.Deps = append(pb.Deps, string(dep))
	//		}
	//	}
	//}

	//return pb, nil
}
