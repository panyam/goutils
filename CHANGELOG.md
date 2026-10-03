# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/). Releases before v0.1.14 were
tag-only and are not back-filled.

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
