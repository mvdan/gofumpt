# gofumpt

`gofmt`, the way it should be pronounced.

	cd $(mktemp -d); go mod init tmp; go get mvdan.cc/gofumpt

Enforce a stricter format than `gofmt`, while being backwards compatible. That
is, `gofumpt` is happy with a subset of the formats that `gofmt` is happy with.

The tool is a modified fork of `gofmt`, so it can be used as a drop-in
replacement. Running `gofmt` after `gofumpt` should be a no-op.

A drop-in replacement for `goimports` is also available:

	cd $(mktemp -d); go mod init tmp; go get mvdan.cc/gofumpt/gofumports

### Features

No empty lines at the beginning or end of a function

<details><summary>example</summary>

```
func foo() {
	println("bar")

}
```

```
func foo() {
	println("bar")
}
```

</details>

No empty lines around a lone statement (or comment) in a block

<details><summary>example</summary>

```
if err != nil {

	return err
}
```

```
if err != nil {
	return err
}
```

</details>

Composite literals with elements in separate lines must also separate both braces

<details><summary>example</summary>

```
var ints = []int{1, 2,
	3, 4}
```

```
var ints = []int{
	1, 2,
	3, 4,
}
```

</details>

Multiline top-level declarations must be separated by empty lines

<details><summary>example</summary>

```
func foo() {
	println("multiline foo")
}
func bar() {
	println("multiline bar")
}
```

```
func foo() {
	println("multiline foo")
}

func bar() {
	println("multiline bar")
}
```

</details>

A single declaration spec must not be grouped with parentheses

<details><summary>example</summary>

```
import (
	"single"
)

var (
	foo = "bar"
)
```

```
import "single"

var foo = "bar"
```

</details>

Comments which aren't Go directives should start with a whitespace

<details><summary>example</summary>

```
//go:noinline

//Foo is awesome.
func Foo() {}
```

```
//go:noinline

// Foo is awesome.
func Foo() {}
```

</details>

### License

Note that much of the code is copied from Go's `gofmt` and `goimports` commands.
You can tell which files originate from the Go repository from their copyright
headers. Their license file is `LICENSE.google`.

`gofumpt`'s original source files are also under the 3-clause BSD license, with
the separate file `LICENSE`.
