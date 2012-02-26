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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// RPCURL returns the url for the AUR's RPC system using the given
// type and arg. url.QueryEscape() is run on both t and arg.
func RPCURL(t, arg string) string {
	t = url.QueryEscape(t)
	arg = url.QueryEscape(arg)

	return "https://aur.archlinux.org/rpc.php?type=" + t + "&arg=" + arg
}

// PKGURL returns the url for the given package with the given
// sub-path.
func PKGURL(pkg, path string) string {
	return "https://aur.archlinux.org/packages/" + pkg[:2] + "/" + pkg + "/" + path
}

// RPCResult represents a response from the AUR's RPC system.
type RPCResult struct {
	Type    string
	Results interface{}
}

// AURInfo retrieves the information about a specific package from
// the AUR. It returns the result and an error, if any.
func AURInfo(name string) (info RPCResult, err error) {
	rsp, err := http.Get(RPCURL("info", name))
	if err != nil {
		return
	}
	defer rsp.Body.Close()

	d := json.NewDecoder(rsp.Body)
	err = d.Decode(&info)
	if err != nil {
		return
	}

	if info.Type == "error" {
		err = &RPCError{
			Type: "info",
			Arg:  name,
			Err:  info.Results.(string),
		}
		return
	}

	return
}

// AURSearch retrieves search results from the AUR using the given
// search. It returns the results and an error, if any.
func AURSearch(arg string) (info RPCResult, err error) {
	rsp, err := http.Get(RPCURL("search", arg))
	if err != nil {
		return
	}
	defer rsp.Body.Close()

	d := json.NewDecoder(rsp.Body)
	err = d.Decode(&info)
	if err != nil {
		return
	}

	if info.Type == "error" {
		err = &RPCError{
			Type: "search",
			Arg:  arg,
			Err:  info.Results.(string),
		}
		return
	}

	return
}

// GetInfo gets the given information from an RPCResult returned by
// AURInfo().
func (r RPCResult) GetInfo(k string) string {
	return r.Results.(map[string]interface{})[k].(string)
}

// GetSearch gets the given information from the given search result
// from an RPCResult returned by AURSearch().
func (r RPCResult) GetSearch(i int, k string) string {
	return r.Results.([]interface{})[i].(map[string]interface{})[k].(string)
}

// GetSourceTar retrieves the source tar for the named package from
// the AUR and returns it as a *tar.Reader. It returns the result and
// nil, or nil and an error, if any.
func GetSourceTar(name string) (*tar.Reader, error) {
	rsp, err := http.Get(PKGURL(name, name+".tar.gz"))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	uz, err := gzip.NewReader(rsp.Body)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, uz)
	if err != nil {
		return nil, err
	}

	r := tar.NewReader(&buf)

	return r, nil
}

// RPCError represents an error returned by the AUR's RPC system.
type RPCError struct {
	Type string // The type that was used in the RPC call.
	Arg  string // The arg that was used in the RPC call.
	Err  string // The text of the error returned by rpc.php.
}

func (err *RPCError) Error() string {
	return "rpc.php?type=" + err.Type + "&" + err.Arg + " returned '" + err.Err + "'"
}
