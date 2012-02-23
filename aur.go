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
	RPCURLFmt = "https://aur.archlinux.org/rpc.php?type=%v&arg=%v"
	PKGURLFmt = "https://aur.archlinux.org/packages/%v"
)

type RPCResult struct {
	Type    string
	Results interface{}
}

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

func (r RPCResult) GetInfo(k string) string {
	return r.Results.(map[string]interface{})[k].(string)
}

func (r RPCResult) GetSearch(i int, k string) string {
	return r.Results.([]interface{})[i].(map[string]interface{})[k].(string)
}

func NewAURPkg(info RPCResult) (*AURPkg, error) {
	rsp, err := http.Get(fmt.Sprintf(PKGURLFmt, info.GetInfo("Name")+"/PKGBUILD"))
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	pb, err := ParsePkgbuild(rsp.Body)
	if err != nil {
		return nil, err
	}

	return &AURPkg{
		info:     info,
		pkgbuild: pb,
	}, nil
}

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
