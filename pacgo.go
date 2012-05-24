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
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"text/tabwriter"
)

// Cmd represents a command.
type Cmd struct {
	// Help is the help line for the command.
	Help string

	// UsageLine is the usage line for the command. For example, in
	//    Usage: pacgo -M [makepkg options]
	// '-M [options]' is the usage line.
	UsageLine string

	// HelpMore is more help text for the command. It is printed
	// when --help <cmd> is run, or if a command gets a bad argument.
	HelpMore string

	// Run is the function that is called when the command is run.
	// The first arg is the command's name that it was registered
	// with, much like how command-line arguments work.
	Run func(...string) error
}

// The registered commands.
var cmds []cmdContext

type cmdContext struct {
	name string
	cmd  *Cmd
}

// RegisterCmd registers the given command for the given arg.
func RegisterCmd(arg string, cmd *Cmd) {
	for i := range cmds {
		if cmds[i].name == arg {
			cmds[i].cmd = cmd
			return
		}
	}

	cmds = append(cmds, cmdContext{arg, cmd})
}

func GetCmd(arg string) *Cmd {
	for _, cmd := range cmds {
		if cmd.name == arg {
			return cmd.cmd
		}
	}

	return nil
}

var (
	// The temporary directory for building AUR packages. Usually
	// /tmp/(arg0)-(uid)
	TmpDir string
)

// MkTmpDir creates a new temporary directory for the given package
// as a subdirectory of TmpDir. It returns the full path of the new
// dir and an error, if any. Note that it always returns the path the
// new dir would have had, even if it fails to create it.
func MkTmpDir(name string) (string, error) {
	tmp := filepath.Join(TmpDir, name)
	err := os.MkdirAll(tmp, 0755)
	if err != nil {
		return tmp, fmt.Errorf("Failed to create %v.", tmp)
	}

	return tmp, nil
}

// Usage prints the usage. If of is the name of a command, Usage()
// prints the help for that command instead.
func Usage(of string) {
	if cmd := GetCmd(of); cmd != nil {
		fmt.Printf("Usage: %v %v\n\n", os.Args[0], cmd.UsageLine)
		fmt.Printf(cmd.HelpMore)
	} else {
		fmt.Printf("Usage: %v <cmd> [options]\n", os.Args[0])

		fmt.Println("Commands:")
		tabw := tabwriter.NewWriter(os.Stdout, 2, 2, 2, ' ', 0)
		for _, cmd := range cmds {
			fmt.Fprintf(tabw, "  %v:\t%v\n", cmd.name, cmd.cmd.Help)
		}
		tabw.Flush()
	}
}

// UsageError represents an error generated by detecting a bad
// argument. Returning one all the way up to main will cause the
// usage to be printed after the error message.
type UsageError struct {
	Arg string
}

func (err *UsageError) Error() string {
	return fmt.Sprintf("Unknown argument: %v", err.Arg)
}

var (
	// PrintUsageError is a special case that causes the usage to be
	// printed without an error message.
	PrintUsageError = &UsageError{""}
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case int:
				os.Exit(r)
			}
		}
	}()

	if filepath.Base(os.Args[0]) == "pacgo.pprof" {
		file, err := os.Create(filepath.Join(os.TempDir(), "pacgo.pprof"))
		if err != nil {
			panic(err)
		}
		defer file.Close()

		err = pprof.StartCPUProfile(file)
		if err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

	if os.Getuid() == 0 {
		Cprintf("[c7]error:[ce] Can't run as root.\n")
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		Usage("")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h", "-help", "--help":
		if len(os.Args) >= 3 {
			Usage(os.Args[2])
		} else {
			Usage("")
		}
		return
	}

	TmpDir = filepath.Join(os.TempDir(), fmt.Sprintf("%v-%v", filepath.Base(os.Args[0]), os.Getuid()))
	err := os.MkdirAll(TmpDir, 0755)
	if err != nil {
		Cprintf("[c7]error:[ce] Failed to create %v.", TmpDir)
		os.Exit(1)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	done := make(chan int)
	go func() {
		if cmd := GetCmd(os.Args[1]); cmd != nil {
			err := cmd.Run(os.Args[1:]...)
			if err != nil {
				if ue, ok := err.(*UsageError); ok {
					if ue != PrintUsageError {
						Cprintf("[c5]%v: [c7]error:[ce] %v\n", os.Args[1], err)
					}
					Usage(os.Args[1])
					os.Exit(2)
				} else {
					Cprintf("[c5]%v: [c7]error:[ce] %v\n", os.Args[1], err)
				}

				done <- 1
				return
			}
		} else {
			Cprintf("[c7]error:[ce] No such command: [c5]%v[ce]\n", os.Args[1])
			Usage("")

			done <- 2
			return
		}
	}()

	select {
	case got := <-sig:
		Cprintf("[c7]error:[ce] Caught [c5]%v[ce]: Exiting.\n", got)
		panic(1)
	case ret := <-done:
		if ret != 0 {
			panic(ret)
		}
	}
}
