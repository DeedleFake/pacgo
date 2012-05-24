package main

type PkgList []Pkg

func (pl PkgList) Len() int {
	return len(pl)
}

func (pl PkgList) Swap(i1, i2 int) {
	pl[i1], pl[i2] = pl[i2], pl[i1]
}

func (pl PkgList) Less(i1, i2 int) bool {
	switch pl[i1].(type) {
	case *LocalPkg:
		if _, ok := pl[i2].(*LocalPkg); !ok {
			return true
		}
	case *PacmanPkg:
		switch pl[i2].(type) {
		case *LocalPkg:
			return false
		case *AURPkg, *PkgbuildPkg:
			return true
		}
	case *AURPkg:
		switch pl[i2].(type) {
		case *LocalPkg, *PacmanPkg:
			return false
		case *PkgbuildPkg:
			return true
		}
	case *PkgbuildPkg:
		if _, ok := pl[i2].(*PkgbuildPkg); !ok {
			return false
		}
	}

	deps := pl[i2].Deps()
	for _, dep := range deps {
		if SamePkg(dep, pl[i1]) {
			return true
		}
	}

	return pl[i1].Name() < pl[i2].Name()
}
