// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

// TODO: replace with the unix build tag once we require Go 1.19 or later
//go:build linux
// +build linux

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	qt "github.com/frankban/quicktest"
	"golang.org/x/sys/unix"
)

func init() {
	// Here rather than in TestMain, to reuse the unix build tag.
	if limit := os.Getenv("TEST_WITH_FILE_LIMIT"); limit != "" {
		n, err := strconv.ParseUint(limit, 10, 64)
		if err != nil {
			panic(err)
		}
		rlimit := unix.Rlimit{Cur: n, Max: n}
		if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &rlimit); err != nil {
			panic(err)
		}
		os.Exit(main1())
	}
}

func TestWithLowOpenFileLimit(t *testing.T) {
	// Safe to run in parallel, as we only change the limit for child processes.
	t.Parallel()

	tempDir := t.TempDir()
	testBinary, err := os.Executable()
	qt.Assert(t, err, qt.IsNil)

	const (
		// Enough directories to run into the ulimit.
		// Enough number of files in total to run into the ulimit.
		numberDirs        = 500
		numberFilesPerDir = 20
		numberFilesTotal  = numberDirs * numberFilesPerDir
	)
	t.Logf("writing %d tiny Go files", numberFilesTotal)
	var allGoFiles []string
	for i := 0; i < numberDirs; i++ {
		// Prefix "p", so the package name is a valid identifier.
		// Add one go.mod file per directory as well,
		// which will help catch data races when loading module info.
		dirName := fmt.Sprintf("p%03d", i)
		dirPath := filepath.Join(tempDir, dirName)
		err := os.MkdirAll(dirPath, 0o777)
		qt.Assert(t, err, qt.IsNil)

		err = os.WriteFile(filepath.Join(dirPath, "go.mod"),
			[]byte(fmt.Sprintf("module %s\n\ngo 1.16", dirName)), 0o666)
		qt.Assert(t, err, qt.IsNil)

		for j := 0; j < numberFilesPerDir; j++ {
			filePath := filepath.Join(dirPath, fmt.Sprintf("%03d.go", j))
			err := os.WriteFile(filePath,
				// Extra newlines so that "-l" prints all paths.
				[]byte(fmt.Sprintf("package %s\n\n\n", dirName)), 0o666)
			qt.Assert(t, err, qt.IsNil)
			allGoFiles = append(allGoFiles, filePath)
		}
	}
	if len(allGoFiles) != numberFilesTotal {
		panic("allGoFiles doesn't have the expected number of files?")
	}
	runGofmt := func(paths ...string) {
		t.Logf("running with %d paths", len(paths))
		cmd := exec.Command(testBinary, append([]string{"-l"}, paths...)...)
		// 256 is a relatively common low limit, e.g. on Mac.
		cmd.Env = append(os.Environ(), "TEST_WITH_FILE_LIMIT=256")
		out, err := cmd.Output()
		var stderr []byte
		if err, _ := err.(*exec.ExitError); err != nil {
			stderr = err.Stderr
		}
		qt.Assert(t, err, qt.IsNil, qt.Commentf("stderr:\n%s", stderr))
		qt.Assert(t, bytes.Count(out, []byte("\n")), qt.Equals, len(allGoFiles))
	}
	runGofmt(tempDir)
	runGofmt(allGoFiles...)
}
