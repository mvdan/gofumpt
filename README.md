# gofumpt

	GO111MODULE=on go get mvdan.cc/gofumpt

Enforce a stricter format than `gofmt`, while being backwards compatible. That
is, `gofumpt` is happy with a subset of the formats that `gofmt` is happy with.

The tool is a modified fork of `gofmt`, so it can be used as a drop-in
replacement. Running `gofmt` after `gofumpt` should be a no-op.

A drop-in replacement for `goimports` is also available:

	GO111MODULE=on go get mvdan.cc/gofumpt/gofumports

Most of the Go source files in this repository belong to the Go project.
The added formatting rules are in the `format` package.

### Added rules

No empty lines at the beginning or end of a function

<details><summary><i>example</i></summary>

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

<details><summary><i>example</i></summary>

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

No empty lines before a simple error check

<details><summary><i>example</i></summary>

```
foo, err := processFoo()

if err != nil {
	return err
}
```

```
foo, err := processFoo()
if err != nil {
	return err
}
```

</details>

Composite literals should use newlines consistently

<details><summary><i>example</i></summary>

```
// A newline before or after an element requires newlines for the opening and
// closing braces.
var ints = []int{1, 2,
	3, 4}

// A newline between consecutive elements requires a newline between all
// elements.
var matrix = [][]int{
	{1},
	{2}, {
		3,
	},
}
```

```
var ints = []int{
	1, 2,
	3, 4,
}

var matrix = [][]int{
	{1},
	{2},
	{
		3,
	},
}
```

</details>

`std` imports must be in a separate group at the top

<details><summary><i>example</i></summary>

```
import (
	"foo.com/bar"

	"io"

	"io/ioutil"
)
```

```
import (
	"io"
	"io/ioutil"

	"foo.com/bar"
)
```

</details>

Short case clauses should take a single line

<details><summary><i>example</i></summary>

```
switch c {
case 'a', 'b',
	'c', 'd':
}
```

```
switch c {
case 'a', 'b', 'c', 'd':
}
```

</details>

Multiline top-level declarations must be separated by empty lines

<details><summary><i>example</i></summary>

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

Single var declarations should not be grouped with parentheses

<details><summary><i>example</i></summary>

```
var (
	foo = "bar"
)
```

```
var foo = "bar"
```

</details>

Contiguous top-level declarations should be grouped together

<details><summary><i>example</i></summary>

```
var nicer = "x"
var with = "y"
var alignment = "z"
```

```
var (
	nicer     = "x"
	with      = "y"
	alignment = "z"
)
```

</details>


Simple var-declaration statements should use short assignments

<details><summary><i>example</i></summary>

```
var s = "somestring"
```

```
s := "somestring"
```

</details>


The `-s` code simplification flag is enabled by default

<details><summary><i>example</i></summary>

```
var _ = [][]int{[]int{1}}
```

```
var _ = [][]int{{1}}
```

</details>


Octal integer literals should use the `0o` prefix on modules using Go 1.13 and later

<details><summary><i>example</i></summary>

```
const perm = 0755
```

```
const perm = 0o755
```

</details>

Comments which aren't Go directives should start with a whitespace

<details><summary><i>example</i></summary>

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

#### Extra rules behind `-extra`

Adjacent parameters with the same type should be grouped together

<details><summary><i>example</i></summary>

```
func Foo(bar string, baz string) {}
```

```
func Foo(bar, baz string) {}
```

</details>

### Installation

`gofumpt` is a replacement for `gofmt`, so you can simply `go get` it as
described at the top of this README and use it.

Alternatively, to use the tool with VS Code, add these settings:

```
"go.formatTool": "goimports",
"go.alternateTools": {
	"goimports": "gofumports",
},
"go.languageServerExperimentalFeatures": {
	"format": false
}
```

You can use `gofmt` instead of `goimports` and `gofumpt` instead of `gofumports`
if you don't need auto-importing on-save.

### Roadmap

This tool is a place to experiment. In the long term, the features that work
well might be proposed for `gofmt` itself.

The tool is also compatible with `gofmt` and is aimed to be stable, so you can
rely on it for your code as long as you pin a version of it.

### License

Note that much of the code is copied from Go's `gofmt` and `goimports` commands.
You can tell which files originate from the Go repository from their copyright
headers. Their license file is `LICENSE.google`.

`gofumpt`'s original source files are also under the 3-clause BSD license, with
the separate file `LICENSE`.
