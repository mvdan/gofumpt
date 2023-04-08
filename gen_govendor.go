// Copyright (c) 2019, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var (
	modulePath = "mvdan.cc/gofumpt"
	vendorDir  = filepath.Join("internal", "govendor")
)

// All the packages which affect the formatting behavior.
var toVendor = []string{
	"go/format",
	"go/printer",
	"go/doc/comment",
	"internal/diff",
}

func main() {
	catch(os.RemoveAll(vendorDir))

	catch(os.MkdirAll(vendorDir, 0o777))
	out, err := exec.Command("go", "env", "GOVERSION").Output()
	catch(err)
	catch(os.WriteFile(filepath.Join(vendorDir, "version.txt"), out, 0o666))

	oldnew := []string{
		"//go:generate", "//disabled go:generate",
	}
	for _, pkgPath := range toVendor {
		oldnew = append(oldnew, pkgPath, path.Join(modulePath, vendorDir, pkgPath))
	}
	replacer := strings.NewReplacer(oldnew...)

	listArgs := append([]string{"list", "-json"}, toVendor...)
	out, err = exec.Command("go", listArgs...).Output()
	catch(err)

	type Package struct {
		Dir        string
		ImportPath string
		GoFiles    []string
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var pkg Package
		err := dec.Decode(&pkg)
		if err == io.EOF {
			break
		}
		catch(err)

		// Otherwise we can't import it.
		dstPkg := strings.TrimPrefix(pkg.ImportPath, "internal/")

		dstDir := filepath.Join(vendorDir, filepath.FromSlash(dstPkg))
		catch(os.MkdirAll(dstDir, 0o777))
		// TODO: if the packages start using build tags like GOOS or GOARCH,
		// we will need to vendor IgnoredGoFiles as well.
		for _, goFile := range pkg.GoFiles {
			srcBytes, err := os.ReadFile(filepath.Join(pkg.Dir, goFile))
			catch(err)

			src := replacer.Replace(string(srcBytes))

			dst := filepath.Join(dstDir, goFile)
			catch(os.WriteFile(dst, []byte(src), 0o666))
		}
	}
}

func catch(err error) {
	if err != nil {
		panic(err)
	}
}
