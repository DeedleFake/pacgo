package main

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

// ExtractTar extracts the contents of tr to the given dir. It
// returns an error, if any.
func ExtractTar(dir string, tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if hdr.Typeflag == tar.TypeDir {
			err = os.Mkdir(filepath.Join(dir, hdr.Name), 0755)
			if err != nil {
				return err
			}
		} else {
			file, err := os.OpenFile(filepath.Join(dir, hdr.Name),
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.FileMode(hdr.Mode),
			)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(file, tr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func init() {
	RegisterCmd("-G", &Cmd{
		Help: "Download the PKGBUILDs and other files for AUR packages.",
		Run: func(args ...string) error {
			var pkgs []string
			for _, arg := range args[1:] {
				if arg[0] == '-' {
					return UsageError{arg}
				} else {
					pkgs = append(pkgs, arg)
				}
			}

			for _, pkg := range pkgs {
				tr, err := GetSourceTar(pkg)
				if err != nil {
					return err
				}

				err = ExtractTar(".", tr)
				if err != nil {
					return err
				}
			}

			return nil
		},
	})
}
