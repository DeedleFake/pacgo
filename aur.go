package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	rsp, err := http.Get(fmt.Sprintf(RPCURLFmt, "info", name))
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

func (r RPCResult) Get(k string) string {
	return r.Results.(map[string]interface{})[k].(string)
}

func NewAURPkg(info RPCResult) (*AURPkg, error) {
	rsp, err := http.Get(fmt.Sprintf(PKGURLFmt, info.Get("Name")+"/PKGBUILD"))
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
