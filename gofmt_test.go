// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"testing"

	"golang.org/x/mod/modfile"
)

func TestShouldSkipPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ignoredPaths []*modfile.Ignore
		path         string
		want         bool
	}{
		"vendor directory": {
			ignoredPaths: nil,
			path:         "vendor",
			want:         true,
		},
		"nested vendor directory": {
			ignoredPaths: nil,
			path:         "vendor",
			want:         true,
		},
		"testdata directory": {
			ignoredPaths: nil,
			path:         "project/testdata",
			want:         true,
		},
		"nested testdata directory": {
			ignoredPaths: nil,
			path:         "project/testdata",
			want:         true,
		},
		"directory in vendor directory": {
			ignoredPaths: nil,
			path:         "project/vendor/dependency",
			want:         false,
		},
		"directory in testdata directory": {
			ignoredPaths: nil,
			path:         "project/testdata/example",
			want:         false,
		},
		"regular directory": {
			ignoredPaths: nil,
			path:         "project/src/main",
			want:         false,
		},
		"ignored path from go.mod": {
			ignoredPaths: []*modfile.Ignore{
				{Path: "generated"},
			},
			path: "generated/code.go",
			want: true,
		},
		"ignored path from go.mod(relative path)": {
			ignoredPaths: []*modfile.Ignore{
				{Path: "./generated"},
			},
			path: "generated/code.go",
			want: true,
		},
		"multiple ignored paths": {
			ignoredPaths: []*modfile.Ignore{
				{Path: "proto"},
				{Path: "mocks"},
			},
			path: "mocks/service.go",
			want: true,
		},
		"path not in ignored list": {
			ignoredPaths: []*modfile.Ignore{
				{Path: "proto"},
			},
			path: "project/service/handler.go",
			want: false,
		},
		"partial match in ignored path": {
			ignoredPaths: []*modfile.Ignore{
				{Path: "gen"},
			},
			path: "project/generator/code.go",
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := shouldSkipPath(tt.ignoredPaths, tt.path)
			if got != tt.want {
				t.Errorf("shouldSkipPath(%v, %q) = %v, want %v", tt.ignoredPaths, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsSubPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		parentPath string
		childPath  string
		want       bool
	}{
		"invalid parent path": {
			parentPath: "//invalid_path",
			childPath:  "/home/user/project",
			want:       false,
		},
		"invalid child path": {
			parentPath: "/home/user",
			childPath:  "//invalid_path",
			want:       false,
		},
		"both invalid paths": {
			parentPath: "//invalid_parent_path",
			childPath:  "//invalid_child_path",
			want:       false,
		},
		"child is subdirectory of parent": {
			parentPath: "/home/user/project",
			childPath:  "/home/user/project/src/main.go",
			want:       true,
		},
		"child is direct subdirectory": {
			parentPath: "/home/user",
			childPath:  "/home/user/project",
			want:       true,
		},
		"same path": {
			parentPath: "/home/user/project",
			childPath:  "/home/user/project",
			want:       true,
		},
		"child is not under parent": {
			parentPath: "/home/user/project1",
			childPath:  "/home/user/project2/file.go",
			want:       false,
		},
		"parent is child of child path": {
			parentPath: "/home/user/project/src/main.go",
			childPath:  "/home/user/project",
			want:       false,
		},
		"partial path match should not be subpath": {
			parentPath: "/home/proj",
			childPath:  "/home/project/file.go",
			want:       false,
		},
		"relative parent path": {
			parentPath: "./src",
			childPath:  "./src/main.go",
			want:       true,
		},
		"relative child path": {
			parentPath: "/home/user/project",
			childPath:  "project/src/main.go",
			want:       false,
		},
		"both relative paths": {
			parentPath: "./project",
			childPath:  "./project/src/main.go",
			want:       true,
		},
		"root path as parent": {
			parentPath: "/",
			childPath:  "/home/user/project",
			want:       true,
		},
		"both has different parent": {
			parentPath: "/etc/foo",
			childPath:  "/home/user",
			want:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := isSubPath(tt.parentPath, tt.childPath)
			if got != tt.want {
				t.Errorf("isSubPath(%q, %q) = %v, want %v", tt.parentPath, tt.childPath, got, tt.want)
			}
		})
	}
}
