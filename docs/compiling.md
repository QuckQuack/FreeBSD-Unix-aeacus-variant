# Compiling aeacus

This document covers everything needed to build `aeacus` from source for Windows, Linux, and FreeBSD, including
release packaging and troubleshooting.

## Do you need to compile it yourself?

**Not necessarily.** The [GitHub Releases page](https://github.com/elysium-suite/aeacus/releases) for the upstream
project publishes pre-built binaries for Windows and Linux for each tagged version. If a release already contains
the binary you need, you can just download it and skip straight to the
[Installation](../README.md#installation) section of the main README.

You only need to compile from source if:

- You're running a platform or architecture that doesn't have a published release binary (**this currently includes
  FreeBSD** -- as of this port, FreeBSD binaries are not yet part of the upstream release process, so you'll need to
  build them yourself, or ask the maintainer of your fork/mirror to publish `aeacus-freebsd.zip` as part of their
  release process).
- You've made local changes to checks, the scoring engine, or anything else in this repository.
- You want to verify the binary you're running was built from source you can inspect (some competitions require
  this for integrity reasons).

If you maintain a fork or mirror of this repository and want your users to avoid compiling it themselves, run
`make release` (or `make release-bsd` for just FreeBSD) and attach the resulting `.zip` files to your own release.

## Requirements

| Tool | Notes |
|---|---|
| [Go](https://go.dev/dl/) | Version 1.19 or newer. Check with `go version`. |
| `git` | To clone the repository. |
| `make` | To use the provided build targets (optional -- see [Compiling without make](#compiling-without-make)). |
| `zip` | Only required for `release`/`release-lin`/`release-win`/`release-bsd` targets. |

> **Note**: Only Linux development environments are officially supported by the upstream project for *running* the
> build tooling (i.e., the machine you compile *on*). A recent Ubuntu VM or container works well. Go's
> cross-compilation support means you can build Windows, Linux, and FreeBSD binaries all from that same Linux
> machine -- you do not need a Windows or FreeBSD machine to produce binaries for those platforms.

## Getting the source

```sh
git clone https://github.com/elysium-suite/aeacus.git
cd aeacus
```

If you're working from a fork or a patch set (such as this FreeBSD port), clone or extract that instead.

## Fetching dependencies

Go modules handle this automatically on your first build, but you can fetch everything up front with:

```sh
go get -v -d -t ./...
```

This reads `go.mod`/`go.sum` and downloads every dependency (including Windows-only packages) to your local module
cache. It only needs to be run once per machine (or after `go.mod` changes) -- normal builds afterward reuse the
cache.

> **Note**: A handful of dependencies (used only by `checks_windows.go` and friends) are Windows-specific. This is
> normal and expected; Go's build constraints (`_windows.go`, `_linux.go`, `_freebsd.go` filename suffixes) mean
> the corresponding code -- and therefore those dependencies -- is only ever *compiled into* a binary when you
> target that platform. They're all still downloaded during `go get`/`go mod tidy` because Go resolves the full
> module graph up front, but that's a one-time network cost.

## Building

`aeacus` actually produces **two** binaries per platform:

- `aeacus` (or `aeacus.exe`) -- the main scoring engine, run on the machine being scored.
- `phocus` (or `phocus.exe`) -- a lightweight client/daemon, built from the same source tree using the `phocus`
  build tag.

The Makefile handles building both together. All commands below are run from the repository root.

### Using make (recommended)

| Target | Platform(s) | Debug symbols? | Output |
|---|---|---|---|
| `make lin` | Linux | No (stripped, `-ldflags '-s -w'`) | `./aeacus`, `./phocus` |
| `make win` | Windows | No | `./aeacus.exe`, `./phocus.exe` |
| `make bsd` | FreeBSD | No | `./aeacus-freebsd`, `./phocus-freebsd` |
| `make lin-dev` | Linux | Yes | `./aeacus`, `./phocus` |
| `make win-dev` | Windows | Yes | `./aeacus.exe`, `./phocus.exe` |
| `make bsd-dev` | FreeBSD | Yes | `./aeacus-freebsd`, `./phocus-freebsd` |
| `make all` | Windows + Linux + FreeBSD | No | all of the above |
| `make all-dev` | Windows + Linux + FreeBSD | Yes | all of the above |

`make all` is the `.DEFAULT_GOAL`, so running plain `make` with no arguments is equivalent to `make all`.

**Example -- build just a stripped FreeBSD binary:**

```sh
make bsd
file ./aeacus-freebsd ./phocus-freebsd
```

Use the `-dev` variants while actively developing/debugging checks; they leave debug symbols in the binary (larger
file size, but usable with `dlv`/`gdb`, and you get real Go panics/stack traces instead of a stripped binary).

### Compiling without make

If you don't have (or don't want to use) `make`, each target is just a couple of `go build` invocations. For
example, to reproduce `make bsd` by hand:

```sh
CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -tags phocus -o ./phocus-freebsd .
CGO_ENABLED=0 GOOS=freebsd go build -ldflags '-s -w' -o ./aeacus-freebsd .
```

Swap `GOOS=freebsd` for `GOOS=linux` or `GOOS=windows` (and drop the `-freebsd` output suffix, using `.exe` for
Windows) to target a different platform. `GOARCH` defaults to `amd64`; add `GOARCH=arm64` etc. if you need a
different architecture (not officially tested, but Go's cross-compilation should handle it for most checks --
anything gated behind `syscall.Sysctl`/`cgo` may need extra verification).

`CGO_ENABLED=0` is important: it produces a statically-linked binary with no dependency on the target's C library,
which matters a lot for FreeBSD/Linux, where the binary may run on a base install without a matching libc/toolchain
available.

## Releases (obfuscated builds)

The `release*` targets are meant for producing binaries you'll actually hand out in a competition, as opposed to
day-to-day development builds. The key difference: they run `misc/dev/gen-crypto.sh` first, which randomizes the
obfuscation keys baked into `crypto.go` (used to scramble check messages and points in the compiled binary, so a
competitor can't just `strings` the binary to find every check). Each release run gets fresh, unique keys.

| Target | Platform(s) | Output |
|---|---|---|
| `make release` | Windows + Linux + FreeBSD | `aeacus-win32.zip`, `aeacus-linux.zip`, `aeacus-freebsd.zip` |
| `make release-lin` | Linux only | `aeacus-linux.zip` |
| `make release-win` | Windows only | `aeacus-win32.zip` |
| `make release-bsd` | FreeBSD only | `aeacus-freebsd.zip` |

Each zip contains the built binaries alongside `assets/`, `misc/`, and `LICENSE` -- everything needed to extract
straight into `/opt/aeacus` (Linux/FreeBSD) or `C:\aeacus\` (Windows) per the main
[Installation](../README.md#installation) instructions.

> **Note**: `crypto.go` is restored to its original (unobfuscated) state automatically at the end of each release
> build (`mv crypto.go.bak crypto.go`), so your working tree stays clean and diff-free. If a release build is
> interrupted partway through, you may find a `crypto.go.bak` left behind -- just `mv crypto.go.bak crypto.go` to
> restore it manually.

**Run a full release build:**

```sh
make release
ls *.zip
# aeacus-win32.zip  aeacus-linux.zip  aeacus-freebsd.zip
```

Then attach those zips to your GitHub release.

## Verifying a build

A few quick sanity checks worth running before you ship a binary, especially after making changes:

```sh
# Static analysis -- should produce no output
go vet ./...

# Formatting -- should produce no output (no files listed)
gofmt -l .

# Confirm the binary is actually built for the platform you expect
file ./aeacus-freebsd
# -> ELF 64-bit LSB executable, x86-64, ... FreeBSD, statically linked, ...

# Run the test suite for the current host platform
go test ./...
```

Cross-compiled test binaries (e.g. running FreeBSD tests from a Linux build machine) can't actually *execute*, since
they're built for a different OS -- but you can still confirm they **compile**, which catches most platform-specific
mistakes:

```sh
GOOS=freebsd GOARCH=amd64 go test -c -o /tmp/freebsd_test.bin .
```

If you have real FreeBSD hardware or a VM available, copy that binary over and run it there for full coverage.

## Troubleshooting

**`go: golang.org/x/sys@... : reading https://golang.org/...: 403/connection refused`**
Some network environments (corporate proxies, sandboxed CI runners, restrictive firewalls) block
`golang.org` directly, since Go's module proxy protocol normally resolves `golang.org/x/...` import paths through
it. Two options:
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
build environments, even when building for Linux or FreeBSD only. It's caused by the environment's module fetch
returning different bytes than what's recorded in `go.sum` (usually because of a proxy/mirror substitution, not an
actual compromised package). If you hit this:
  - Try building with a normal, unrestricted internet connection first to confirm it's environment-specific.
  - As a last resort in a locked-down CI environment, you can regenerate `go.sum` (`rm go.sum && go mod tidy`), but
    treat that as a signal to investigate *why* your environment disagrees with the published checksums --
    regenerating `go.sum` should not be your default fix.

**Build succeeds but the binary won't run on the target machine**
Double-check `CGO_ENABLED=0` was set. A `CGO_ENABLED=1` build links against the *build machine's* C library, which
usually doesn't exist (or doesn't match) on the target machine, especially across OSes.

**`make: command not found`**
Install `make` via your distro's package manager (`apt install make` on Ubuntu/Debian), or just run the equivalent
`go build` command by hand -- see [Compiling without make](#compiling-without-make).

**Windows build fails with missing `golang.org/x/sys/windows/...` or similar**
Make sure you ran `go get -v -d -t ./...` at least once so the Windows-only dependencies are present in your module
cache -- they aren't downloaded just by building for Linux/FreeBSD.
