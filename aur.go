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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	// The URL format for using the AUR's RPC system.
	RPCURLFmt = "https://aur.archlinux.org/rpc.php?type=%v&arg=%v"

	// The URL format for downloading stuff from the AUR.
	PKGURLFmt = "https://aur.archlinux.org/packages/%v"
)

// RPCResult represents a response from the AUR's RPC system.
type RPCResult struct {
	Type    string
	Results interface{}
}

// AURInfo retrieves the information about a specific package from
// the AUR. It returns the result and an error, if any.
func AURInfo(name string) (info RPCResult, err error) {
	rsp, err := http.Get(fmt.Sprintf(RPCURLFmt, "info", url.QueryEscape(name)))
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
		return info, errors.New(info.Results.(string))
	}

	return
}

// AURSearch retrieves search results from the AUR using the given
// search. It returns the results and an error, if any.
func AURSearch(arg string) (info RPCResult, err error) {
	rsp, err := http.Get(fmt.Sprintf(RPCURLFmt, "search", url.QueryEscape(arg)))
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
		return info, errors.New(info.Results.(string))
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
	rsp, err := http.Get(fmt.Sprintf(PKGURLFmt, name+"/"+name+".tar.gz"))
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
