# AGENTS.md

This file provides guidance to coding agents such as Claude Code (claude.ai/code)
when working with code in this repository. `CLAUDE.md` is a symlink to it.

## Commands

```sh
go test ./...                       # all tests
go test -run TestScript/short-decl  # one testscript (name = testdata/script/<name>.txtar)
go test -run TestScript/short-decl -u   # rewrite the txtar's expected output in place
go test -run 'TestScript/^$' -fuzz FuzzFormat ./format
go vet ./...
gofmt -s -d .                       # CI requires no output
```

CI runs `go test ./...` and `go test -race ./...` on the last two Go minor versions, on Linux/macOS/Windows.

`go generate ./...` re-vendors Go's formatting packages: `gen_govendor.go` copies `go/format`,
`go/printer`, `go/doc/comment` and `internal/diff` from the toolchain in `PATH` into
`internal/govendor` (rewriting import paths and recording the version in `internal/govendor/version.txt`),
then `go run . -w internal/govendor` re-formats them with the freshly built gofumpt.
Only do this when deliberately bumping the vendored Go version.

## Architecture

gofumpt is a fork of `cmd/gofmt`. The split matters when editing:

- **Inherited from upstream Go, updated by hand**: `gofmt.go`, `internal.go`,
  `format/rewrite.go`, `format/simplify.go`. Keep them close to upstream so future
  merges stay tractable; gofumpt-specific edits are marked with `NOTE(gofumpt)` comments.
  `rewrite.go`/`simplify.go` live under `format` because simplification is part of the public API.
- **Frozen toolchain copies**: `internal/govendor/...`. Never edit by hand — regenerate.
  These exist so a given gofumpt version formats identically regardless of the Go version
  used to build it.
- **gofumpt's own rules**: `format/format.go`, plus the public `format.Source`/`format.File`
  and `format.Options` API.

`format.File` runs `simplify` (gofmt's `-s`, always on), then walks the AST with
`astutil.Apply`. `fumpter.applyPre`/`applyPost` implement the rules; a `pre`/`post` wrapper
tracks contextual state (`blockLevel`, `parentFuncTypes`, `minSplitFactor` for func params vs results).

Most rules do not rewrite syntax — they move newlines. A `fumpter` holds the `token.File` and
manipulates line offsets directly (`addNewline`, `removeLines`, `removeLinesBetween`, `lineEnd`),
since `go/printer` derives blank lines from the fset. `printLength` renders a node to count bytes
for the `shortLineLimit` / `longLineLimit` / `minSplitFactor` heuristics.

Behavior can depend on `Options.LangVersion` (e.g. `0o` octal literals need go1.13) and
`Options.ModulePath` (deciding what counts as a std import). The CLI derives both from the
nearest `go.mod` (`loadModule`, cached per directory), overridable with `-lang` and `-modpath`.
`go.mod` `ignore` directives, `vendor`/`testdata` dirs, and generated files are skipped unless
passed as explicit arguments.

Extra rules are opt-in via `-extra`; `format.Extra` is a struct of named booleans with a
string-based `Set` so adding or removing one doesn't break Go API users.

## Conventions

- Output must be gofmt-stable: running `gofmt` after `gofumpt` produces no changes.
  gofumpt only ever narrows the set of formats gofmt accepts.
- Rules must be idempotent in a single pass. Testscripts enforce this with the standard shape:
  format the input, `cmp` against `.golden`, then run `gofumpt -d` on the golden and assert no output.
- Tests are `testscript` archives in `testdata/script/*.txtar` (`RequireExplicitExec` is on, so
  commands need `exec`). The fuzz corpus is seeded from the `.go` files inside them.
- A new or changed rule needs a testscript and a README entry under "Added rules" (or
  "Extra rules behind `-extra`") in the same commit.

## Fuzz failures

`FuzzFormat` checks that gofumpt is idempotent, that gofmt does not change its output, and that
no identifiers, literals or comment words are added or removed. To find and resolve one failure:

1. Run `go test -run='^$' -fuzz=FuzzFormat -fuzztime=10m -timeout=12m ./format`.
   Keep `-timeout` above `-fuzztime`, or the test binary is killed mid-run.
   It stops at the first failure, writing the input to `format/testdata/fuzz/FuzzFormat/<hash>`;
   re-run it alone with `go test -run=FuzzFormat/<hash> ./format`.
2. Reduce it to a small file and reproduce it with a built gofumpt: format it, then run
   `gofumpt -d` or `gofmt -d` on the result. Find which rule's line or comment handling
   disagrees with what go/printer does on the next run.
3. If real code could plausibly hit it, fix it in one commit with a minimal case in the testscript
   for that rule, such as `0f62a1d`. Unlike other bug fixes, it is not split into a test commit:
   `FuzzFormat` seeds its corpus from the testscripts, so an unstable case fails until the fix.
4. If the input is very unlikely in real code, or the bug is in `internal/govendor`, skip it
   instead: add a check to `fuzzUnsupported` in `format/fuzz_test.go`, with a comment on why,
   plus a TODO naming when to remove it if a vendored Go update fixes it. One commit, such as
   `format: skip runs of backticks in comments when fuzzing`.
5. Do not commit the crasher file: the corpus is seeded from the testscripts, which cover a fix,
   and a skipped input would only be skipped again.

## Commit practices

Drawn from the last 200 commits.

- Subject: lowercase `package/path: what changed`, imperative, ~40 chars and rarely over 70.
  The prefix is the directory or file touched — `format:`, `testdata:`, `README:`,
  `CHANGELOG:`, `internal/govendor:` — omitted only for repo-wide changes (`update deps`).
- Three quarters of commits have a body, and non-trivial ones always do. The shape is:
  a paragraph explaining the *mechanism* of the bug (which function, which wrong assumption,
  why the output was wrong), then a short paragraph naming the fix, then `Fixes #123.`
  (or `Updates #123.` when the issue stays open). Explain why, not what the diff shows.
- Bug fixes land as two commits: first a test-only commit adding the failing case to a
  testscript, whose golden file records the *current, wrong* output plus a `# TODO:` header
  comment describing the intended behavior — it passes on the unfixed code. Then the fix
  commit, which flips the golden file and drops the TODO. See `360940c` / `05e0adf`.
  Formatting that is not stable is the exception; see "Fuzz failures" above.
- A `format/format.go` change nearly always touches a `testdata/script/*.txtar` in the same
  commit; prefer extending the existing script for that rule over adding a new one.
- CHANGELOG entries are written at release time under a new `## [vX.Y.Z] - DATE` heading,
  or added directly when an `## Unreleased` heading exists — not per commit.
- Vendored Go updates are their own commit (`internal/govendor: \`go generate\` with Go 1.27.0`),
  paired with a CI matrix bump (`add Go 1.27.x, drop 1.25.x`) and separate from `update deps`.
