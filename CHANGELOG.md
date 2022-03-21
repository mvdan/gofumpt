# Changelog

## [v0.3.1] - 2022-03-21

This bugfix release resolves a number of issues:

* Avoid "too many open files" error regression introduced by [v0.3.0] - [#208]
* Use the `go.mod` relative to each Go file when deriving flag defaults - [#211]
* Remove unintentional debug prints when directly formatting files

## [v0.3.0] - 2022-02-22

This is gofumpt's third major release, based on Go 1.18's gofmt.
The jump from Go 1.17's gofmt should bring a noticeable speed-up,
as the tool can now format many files concurrently.
On an 8-core laptop, formatting a large codebase is 4x as fast.

The following [formatting rules](https://github.com/mvdan/gofumpt#Added-rules) are added:

* Functions should separate `) {` where the indentation helps readability
* Field lists should not have leading or trailing empty lines

The following changes are included as well:

* Generated files are now fully formatted when given as explicit arguments
* Prepare for Go 1.18's module workspaces, which could cause errors
* Import paths sharing a prefix with the current module path are no longer
  grouped with standard library imports
* `format.Options` gains a `ModulePath` field per the last bullet point

## [v0.2.1] - 2021-12-12

This bugfix release resolves a number of issues:

* Add deprecated flags `-s` and `-r` once again, now giving useful errors
* Avoid a panic with certain function declaration styles
* Don't group interface members of different kinds
* Account for leading comments in composite literals

## [v0.2.0] - 2021-11-10

This is gofumpt's second major release, based on Go 1.17's gofmt.
The jump from Go 1.15's gofmt should bring a mild speed-up,
as walking directories with `filepath.WalkDir` uses fewer syscalls.

gofumports is now removed, after being deprecated in [v0.1.0].
Its main purpose was IDE integration; it is now recommended to use gopls,
which in turn implements goimports and supports gofumpt natively.
IDEs which don't integrate with gopls (such as GoLand) implement goimports too,
so it is safe to use gofumpt as their "format on save" command.
See the [installation instructions](https://github.com/mvdan/gofumpt#Installation)
for more details.

The following [formatting rules](https://github.com/mvdan/gofumpt#Added-rules) are added:

* Composite literals should not have leading or trailing empty lines
* No empty lines following an assignment operator
* Functions using an empty line for readability should use a `) {` line instead
* Remove unnecessary empty lines from interfaces

Finally, the following changes are made to the gofumpt tool:

* Initial support for Go 1.18's type parameters is added
* The `-r` flag is removed in favor of `gofmt -r`
* The `-s` flag is removed as it is always enabled
* Vendor directories are skipped unless given as explicit arguments
* The added rules are not applied to generated Go files
* The `format` Go API now also applies the `gofmt -s` simplification
* Add support for `//gofumpt:diagnose` comments

## [v0.1.1] - 2021-03-11

This bugfix release backports fixes for a few issues:

* Keep leading empty lines in func bodies if they help readability
* Avoid breaking comment alignment on empty field lists
* Add support for `//go-sumtype:` directives

## [v0.1.0] - 2021-01-05

This is gofumpt's first release, based on Go 1.15.x. It solidifies the features
which have worked well for over a year.

This release will be the last to include `gofumports`, the fork of `goimports`
which applies `gofumpt`'s rules on top of updating the Go import lines. Users
who were relying on `goimports` in their editors or IDEs to apply both `gofumpt`
and `goimports` in a single step should switch to gopls, the official Go
language server. It is supported by many popular editors such as VS Code and
Vim, and already bundles gofumpt support. Instructions are available [in the
README](https://github.com/mvdan/gofumpt).

`gofumports` also added maintenance work and potential confusion to end users.
In the future, there will only be one way to use `gofumpt` from the command
line. We also have a [Go API](https://pkg.go.dev/mvdan.cc/gofumpt/format) for
those building programs with gofumpt.

Finally, this release adds the `-version` flag, to print the tool's own version.
The flag will work for "master" builds too.

[v0.3.1]: https://github.com/mvdan/gofumpt/releases/tag/v0.3.1
[#208]: https://github.com/mvdan/gofumpt/issues/208
[#211]: https://github.com/mvdan/gofumpt/pull/211

[v0.3.0]: https://github.com/mvdan/gofumpt/releases/tag/v0.3.0
[v0.2.1]: https://github.com/mvdan/gofumpt/releases/tag/v0.2.1
[v0.2.0]: https://github.com/mvdan/gofumpt/releases/tag/v0.2.0
[v0.1.1]: https://github.com/mvdan/gofumpt/releases/tag/v0.1.1
[v0.1.0]: https://github.com/mvdan/gofumpt/releases/tag/v0.1.0
