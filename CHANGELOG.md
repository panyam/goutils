# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/). Releases before v0.1.14 were
tag-only and are not back-filled.

## [Unreleased]

### Removed

- `experimental/gae`, the Datastore helper. It didn't build against grpc 1.82, CI skipped it, and
  nothing imports it (#6, #3). Its removal drops cloud.google.com/go, google.golang.org/api and the
  rest of the Google Cloud modules from goutils's requires.
- `grpc`, the `ErrorLogger` interceptor and the protojson marshaller. Nothing imports them, and they
  were the only reason goutils required grpc (#6).

### Changed

- `go 1.26.0` is the floor for importers, and goutils builds on `toolchain go1.27.4`.
- A weekly workflow scans for known vulnerabilities through `panyam/ci-actions/vulncheck`
  (panyam/ci-actions#11).

## [v0.1.15] - 2026-11-19

### Added

- `mountfs`: `FS.ContentSum` asks the mount that holds a file for its content sum, when that
  mount's `fs.FS` has a `ContentSum(name) (sum string, size int64, ok bool)` method, as a store
  that keeps its files by hash does. A caller that fingerprints files takes the sum rather than
  reading the file, through the composed tree. agni's wasm engine needs it to replace its own copy
  of `mountfs` (panyam/agni#1761).

## [v0.1.14] - 2026-10-03

### Added

- `memfs`: a concurrency-safe in-memory `fs.FS`, built on `fstest.MapFS`. Files
  can be added, replaced or removed while other goroutines read, and a file
  already open keeps its contents. It has os-style writes (`WriteFile`,
  `MkdirAll`, `Remove`, `Rename`) that satisfy templar's `WritableFS`, and
  `Put`/`New` take ownership of the bytes so a wasm host avoids a second copy.
  (PR 4)
- `mountfs`: composes named `fs.FS` values into one `fs.FS` whose top-level
  directories are the mounts, so a file in one mount can name a file in another
  by path. `Mount` and `Unmount` change the table under concurrent readers.
  (PR 4)
- CI on pull requests and master: `go test -race`, plus `memfs` and `mountfs`
  run as wasm under Node. (PR 4)

### Changed

- Dependency bumps from dependabot (PR 1), including grpc.

### Known issues

- `experimental/gae` does not build against the bumped grpc, so CI skips it
  (`goutils#3`). No other package is affected.
