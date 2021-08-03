package utils

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// IsAbsolutePath returns if path is an absolute path in *nix and Windows file systems
func IsAbsolutePath(p string) bool {
	winAbsolutePathPrefix := regexp.MustCompile(`[A-Z]:`)
	loc := winAbsolutePathPrefix.FindStringIndex(p)
	return path.IsAbs(p) || (loc != nil && loc[0] == 0 && loc[1] == 2)
}

// DeriveStaticURIFromPath takes a string representing a file system path and its optional alias, and returns the path
// itself and the leaf segment of the path prepended with a forward slash if it does not exist. If the input is empty,
// just a forward slash will be returned. See the following examples:
//
// 1. folder => /folder
// 2. /folder => /folder
// 3. folder:my-folder => /my-folder
// 4. folder:/my-folder => /my-folder
// 5. nested/project => /project
// 6. nested/project:my-project => /my-project

func DeriveStaticURIFromPath(input string) (string, string) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return input, "/"
	}

	split := strings.Split(input, ":")
	p := split[0]
	uri := ""

	if len(split) > 1 {
		if !strings.HasPrefix(split[1], "/") {
			uri += "/"
		}
		uri += split[1]
	} else {
		if !strings.HasPrefix(split[0], "/") {
			uri += "/"
		}
		uri += filepath.Base(split[0])
	}

	return p, uri
}

func JoinBasePathIfRelativeRegularFilePath(base string, in string) (out string) {
	out = in
	if in == "stdout" || in == "stderr" {
		return
	}

	if !IsAbsolutePath(in) {
		out = filepath.Join(base, out)
	}
	return
}
