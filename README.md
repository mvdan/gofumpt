# gofumpt

`gofmt`, the way it should be pronounced.

	cd $(mktemp -d); go mod init tmp; go get mvdan.cc/gofumpt

Enforce a stricter format than `gofmt`, while being backwards compatible. That
is, `gofumpt` is happy with a subset of the formats that `gofmt` is happy with.

The tool is a modified fork of `gofmt`, so it can be used as a drop-in
replacement. Running `gofmt` after `gofumpt` should be a no-op.

### Features

No empty lines at the beginning or end of a function:

```
func foo() {
	println("bar")

}
```

No empty lines around a lone statement (or comment) in a block:

```
if err != nil {

	return err
}
```

Composite literals with newlines between elements must also separate the opening
and closing braces with newlines:


```
var bad = []int{1, 2,
	3, 4}

var good = []int{
	1, 2,
	3, 4,
}
```

Multiline top-level declarations must be separated by empty lines:

```
func foo() {
	println("multiline foo")
}
func bar() {
	println("multiline bar")
}
```

### License

Note that much of the code is copied from Go's `cmd/gofmt` command. You can tell
which files originate from the Go repository from their copyright headers. Their
license file is `LICENSE.google`.

`gofumpt`'s original source files are also under the 3-clause BSD license, with
the separate file `LICENSE`.
