# Compiling aeacus for FreeBSD

This document covers building `aeacus` from source for **FreeBSD**.

## Do you need to compile it yourself?

You don’t have to, but it’s highly recommended. The [GitHub Releases page](https://github.com/elysium-suite/aeacus/releases) for the
upstream project publishes pre-built binaries for Windows and Linux -- FreeBSD binaries are not yet part of the
upstream release process. To get a FreeBSD binary you'll need to either build it yourself.
Update: I have now added `aeacus-freebsd.zip` as a pre compiled version; though it uses the same hash that any other copy would have so it’s therefore less secure. 



## Requirements

| Tool | Notes |
|---|---|
| [Go](https://go.dev/dl/) | Version 1.19 or newer. Check with `go version`. |
| `git` | To clone the repository. (or just download the tar.zip file) |
| `make` | To use the provided build targets (optional -- see [Compiling without make](#compiling-without-make)). |
| `zip` | Only required for the `release-bsd` target. |

> **Note**: You do not need actual FreeBSD hardware to *build* the binary. Only Linux development environments are
> officially supported by the upstream project for running the build tooling -- a recent Ubuntu VM or container
> works well. Go's cross-compilation support lets you produce a FreeBSD binary from that same Linux machine. You
> will, however, want real FreeBSD hardware or a VM to actually *run and test* the resulting binary.

## Getting the source

```sh
git clone https://github.com/elysium-suite/aeacus.git
cd aeacus
```



## Fetching dependencies

Go modules handle this automatically on your first build, but you can fetch everything up front with:

```sh
go get -v -d -t ./...
```

This reads `go.mod`/`go.sum` and downloads every dependency to your local module cache. It only needs to be run
once per machine (or after `go.mod` changes) -- normal builds afterward reuse the cache. Note that this also
downloads Windows only dependencies (needed elsewhere in the module graph); that's expected and harmless for a
FreeBSD build.

## Building

`aeacus` produces **two** binaries:

- `aeacus-freebsd` -- the main scoring engine, run on the machine being scored.
- `phocus-freebsd` -- a lightweight client/daemon, built from the same source tree using the `phocus` build tag.

The Makefile handles building both together. All commands below are run from the repository root.

### Using make (recommended)

| Target | Debug symbols? | Output |
|---|---|---|
| `make bsd` | No (stripped, `-ldflags '-s -w'`) | `./aeacus-freebsd`, `./phocus-freebsd` |
| `make bsd-dev` | Yes | `./aeacus-freebsd`, `./phocus-freebsd` |

Use the `-dev` variant while actively developing/debugging checks; it leaves debug symbols in the binary (larger
file size, but usable with `dlv`/`gdb`, and you get real Go panics/stack traces instead of a stripped binary).

**Build a stripped FreeBSD binary:**

```sh
make bsd
file ./aeacus-freebsd ./phocus-freebsd
```

### Compiling without make

If you don't have (or don't want to use) `make`, the target is just a couple of `go build` invocations:

```sh
CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -tags phocus -o ./phocus-freebsd .
CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -o ./aeacus-freebsd .
```

Drop `-ldflags '-s -w'` for a dev build with debug symbols. `GOARCH` defaults to `amd64`; add `GOARCH=arm64` etc.
if you need a different architecture (not officially tested, but Go's cross-compilation should handle it for most
checks -- anything gated behind `syscall.Sysctl`/`cgo` may need extra verification).

`CGO_ENABLED=0` is important: it produces a statically-linked binary with no dependency on the target's C library,
which matters a lot for FreeBSD, where the binary may run on a base install without a matching libc/toolchain
available.

## Releases (obfuscated builds)

`make release-bsd` is meant for producing a binary you'll actually hand out in a competition, as opposed to a
day-to-day development build. The key difference: it runs `misc/dev/gen-crypto.sh` first, which randomizes the
obfuscation keys baked into `crypto.go` (used to scramble check messages and points in the compiled binary, so a
competitor can't just `strings` the binary to find every check). Each release run gets fresh, unique keys.

| Target | Output |
|---|---|
| `make release-bsd` | `aeacus-freebsd.zip` |

The zip contains the built binaries alongside `assets/`, `misc/`, and `LICENSE` -- everything needed to extract
straight into `/opt/aeacus` per the main [Installation](../README.md#installation) instructions.

> **Note**: `crypto.go` is restored to its original (unobfuscated) state automatically at the end of the release
> build (`mv crypto.go.bak crypto.go`), so your working tree stays clean and diff-free. If a release build is
> interrupted partway through, you may find a `crypto.go.bak` left behind -- just `mv crypto.go.bak crypto.go` to
> restore it manually.

**Run a full release build:**

```sh
make release-bsd
ls *.zip
# aeacus-freebsd.zip
```

Then attach the zip to your GitHub release.

## Verifying a build

A few quick sanity checks worth running before you ship a binary, especially after making changes:

```sh
# Static analysis -- should produce no output
go vet ./...

# Formatting -- should produce no output (no files listed)
gofmt -l .

# Confirm the binary is actually built for FreeBSD
file ./aeacus-freebsd
# -> ELF 64-bit LSB executable, x86-64, ... FreeBSD, statically linked, ...
```

Since the build machine is Linux, the cross-compiled binary can't actually *execute* there -- but you can still
confirm the test suite **compiles** for FreeBSD, which catches most platform-specific mistakes:

```sh
GOOS=freebsd GOARCH=amd64 go test -c -o /tmp/freebsd_test.bin .
```

If you have real FreeBSD hardware or a VM available, copy that binary over and run it there for full coverage.

## Troubleshooting

**`go: golang.org/x/sys@... : reading https://golang.org/...: 403/connection refused`**
Some network environments (corporate proxies, sandboxed CI runners, restrictive firewalls) block `golang.org`
directly, since Go's module proxy protocol normally resolves `golang.org/x/...` import paths through it. Two
options:
  - Point `GOPROXY` at the default public proxy explicitly: `GOPROXY=https://proxy.golang.org,direct go build ...`
    (this is usually already the default -- the issue is typically that *this specific host* is blocked, not the
    Go proxy protocol itself).
  - If `golang.org` itself is blocked but `github.com` is reachable, add a `replace` directive to route the mirror:
    ```
    replace golang.org/x/sys => github.com/golang/sys v0.0.0-20220829200755-d48e67d00261
    ```
    (Match the version to whatever `go.mod` currently requires.) Remove this line before committing/releasing --
    it's a local workaround, not something that should ship.

**`SECURITY ERROR ... checksum mismatch` for an unrelated dependency**
This has been observed for Windows-only dependencies (e.g. `github.com/StackExchange/wmi`) in some sandboxed/offline
build environments, even when building for FreeBSD only. It's caused by the environment's module fetch returning
different bytes than what's recorded in `go.sum` (usually because of a proxy/mirror substitution, not an actual
compromised package). If you hit this:
  - Try building with a normal, unrestricted internet connection first to confirm it's environment-specific.
  - As a last resort in a locked-down CI environment, you can regenerate `go.sum` (`rm go.sum && go mod tidy`), but
    treat that as a signal to investigate *why* your environment disagrees with the published checksums --
    regenerating `go.sum` should not be your default fix.

**Build succeeds but the binary won't run on the target machine**
Double-check `CGO_ENABLED=0` was set. A `CGO_ENABLED=1` build links against the *build machine's* C library, which
usually doesn't exist (or doesn't match) on the FreeBSD target machine.

**`make: command not found`**
Install `make` via your distro's package manager (`apt install make` on Ubuntu/Debian), or just run the equivalent
`go build` command by hand -- see [Compiling without make](#compiling-without-make).
