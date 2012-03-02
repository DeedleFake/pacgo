package main

import (
	"fmt"
	"os"
	"os/exec"
)

func init() {
	RegisterCmd("-V", &Cmd{
		Help:      "Show pacgo's version.",
		UsageLine: "-V",
		HelpMore: `-V shows the version of pacgo. For git versions, it shows the time of
compilation.
`,
		Run: func(args ...string) error {
			// TODO: Figure out a better way to do this...

			file, err := os.Stat(os.Args[0])
			if err != nil {
				path, err := exec.LookPath(os.Args[0])
				if err != nil {
					return fmt.Errorf("Can't find %v.", os.Args[0])
				}
				file, err = os.Stat(path)
				if err != nil {
					return fmt.Errorf("Can't open %v.", path)
				}
			}

			fmt.Println(file.ModTime())

			return nil
		},
	})
}
