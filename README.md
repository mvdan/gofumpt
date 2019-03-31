# gofumpt

`gofmt`, the way it should be pronounced.

Enforce a stricter format than `gofmt`, while still being compatible with
`gofmt`. That is, `gofumpt` is happy with a subset of the formats that `gofmt`
is happy with.

The tool is a modified fork of `gofmt`, so it can be used as a drop-in
replacement.

### Features

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

### License

Note that much of the code is copied from Go's `cmd/gofmt` command. You can tell
which files originate from the Go repository from their copyright headers.
`gofumpt`'s original source files are also under the 3-clause BSD license, hence
there's only one LICENSE file.
