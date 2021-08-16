// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"io"
	"net/http"
	"os"
)

type NoDirFileSystem struct {
	fs http.Dir
}

type neuteredStatFile struct {
	http.File
	readDirCount int
}

func (e neuteredStatFile) Stat() (os.FileInfo, error) {
	s, err := e.File.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
	LOOP:
		for {
			fl, err := e.File.Readdir(e.readDirCount)
			switch err {
			case io.EOF:
				break LOOP
			case nil:
				for _, f := range fl {
					if f.Name() == "index.html" {
						return s, err
					}
				}
			default:
				return nil, err
			}
		}
		return nil, os.ErrNotExist
	}
	return s, err
}

func (nd NoDirFileSystem) Open(name string) (http.File, error) {
	f, err := nd.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return neuteredStatFile{f, 2}, nil
}
