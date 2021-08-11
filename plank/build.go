// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"bufio"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var (
	GOOS string
	GOARCH string
	versionString string
	outputPath string
	LDFLAGS = []string{"-s", "-w"}
)

// main application to build Plank
// specify BUILD_OS and BUILD_ARCH that map to GOOS and GOARCH to support building against
// a platform different from the caller's OS. output will be generated at build/
func main() {
	// read build envrionemnt variables like GOOS and GOARCH
	GOOS = os.Getenv("BUILD_OS")
	GOARCH = os.Getenv("BUILD_ARCH")
	goosSelected := GOOS
	goarchSelected := GOARCH
	if len(goosSelected) == 0 {
		goosSelected = runtime.GOOS
	}

	if len(goarchSelected) == 0 {
		goarchSelected = runtime.GOARCH
	}

	WD, _ := os.Getwd()
	r, err := cloneCurrentBranch(WD)
	if err != nil {
		log.Fatalln(err)
	}


	// assemble version string
	// if it's a branch or tag, version string should be ${BRANCH}-${HASH} otherwise ${HASH}
	branch, err := getBranch(r)
	if err != nil {
		log.Fatalln(err)
	}

	tags, err := getTags(r)
	if err != nil {
		log.Fatalln(err)
	}

	versionString = buildVersionString(branch, tags)
	LDFLAGS = append(LDFLAGS, "-X main.version=" + versionString)

	// set output path
	binaryExt := ""
	if goosSelected == "windows" {
		binaryExt = ".exe"
	}
	outputPath = filepath.Join(WD, "build", "plank"+binaryExt)

	fmt.Printf("Building Plank\n")
	fmt.Println()
	fmt.Printf("GOOS\t\t%s\n", goosSelected)
	fmt.Printf("GOARCH\t\t%s\n", goarchSelected)
	fmt.Printf("Version\t\t%s\n", versionString)
	fmt.Printf("Output\t\t%s\n", outputPath)
	fmt.Printf("ldflags\t\t%v\n", LDFLAGS)

	buildCommand := []string{"build", "-o", outputPath, "-ldflags", "'"+strings.Join(LDFLAGS, " ")+"'", "cmd/main.go"}
	fmt.Println()
	fmt.Printf("Build command:\ngo %s\n", strings.Join(buildCommand, " "))

	cmd := exec.Command("go",
		"build", "-o", outputPath, "-ldflags", strings.Join(LDFLAGS, " "), "cmd/main.go")
	cmd.Env = append(os.Environ(), "GOOS="+goosSelected, "GOARCH="+goarchSelected)
	cmd.Dir = WD
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	outScanner := bufio.NewScanner(stdout)
	errScanner := bufio.NewScanner(stderr)
	outScanner.Split(bufio.ScanLines)
	errScanner.Split(bufio.ScanLines)

	wg := sync.WaitGroup{}
	wg.Add(2)
	err = cmd.Start()
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for outScanner.Scan() {
			txt := outScanner.Text()
			fmt.Println(txt)
		}
		wg.Done()
	}()

	go func() {
		for errScanner.Scan() {
			txt := errScanner.Text()
			fmt.Println(txt)
		}
		wg.Done()
	}()

	wg.Wait()
	state, err := cmd.Process.Wait()
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println()
	if state.ExitCode() == 0 {
		log.Println("Build successful")
	} else {
		log.Println("Build failed")
	}
	os.Exit(state.ExitCode())
}

// cloneCurrentBranch clones the current branch into memory and return the reference
func cloneCurrentBranch(wd string) (*git.Repository, error) {
	if filepath.Base(wd) != "plank" {
		return nil, fmt.Errorf("this build application needs to be run inside plank/ folder")
	}

	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: filepath.Clean(filepath.Join(wd, "..")),
	})
	return r, err
}

// getTags retrieves all tags and sort by recent versions
func getTags(r *git.Repository) ([]string, error) {
	sortedTags := make([]string, 0)
	tags, err := r.Tags()
	if err != nil {
		return nil, err
	}

	_ = tags.ForEach(func(ref *plumbing.Reference) error {
		sortedTags = append(sortedTags, ref.Name().Short())
		return nil
	})

	// sort tags
	sort.SliceStable(sortedTags, func(i, j int) bool {
		aExploded := strings.Split(sortedTags[i], ".")
		bExploded := strings.Split(sortedTags[j], ".")
		majorA := aExploded[0]
		majorB := bExploded[0]
		minorA := aExploded[1]
		minorB := bExploded[1]
		patchA := aExploded[2]
		patchB := bExploded[2]

		if majorA > majorB {
			return true
		} else if majorA < majorB {
			return false
		}

		if minorA > minorB {
			return true
		} else if minorA < minorB {
			return false
		}

		if patchA > patchB {
			return true
		}
		return false
	})
	return sortedTags, nil
}

func getBranch(r *git.Repository) (*plumbing.Reference, error) {
	branches, err := r.Branches()
	if err != nil {
		return nil, err
	}
	defer branches.Close()

	return branches.Next()
}

// buildVersionString takes the git object reference and tags as inputs and assembles a version string based on it
// being a branch/tag or generic object references
func buildVersionString(ref *plumbing.Reference, tags []string) (str string) {
	if len(tags) > 0 {
		str = tags[0] + "-"
	}
	if ref.Name().IsBranch() || ref.Name().IsTag() {
		str += ref.Name().Short() + "-" + ref.Hash().String()[:8]
	} else {
		str += ref.Hash().String()[:8]
	}
	return str
}