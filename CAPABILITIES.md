# goutils

## Version
0.1.10

## Provides
- concurrency-utilities: Reader, Writer, Mapper, Reducer with generics
- stream-processing: Pipe, FanIn, FanOut patterns
- thread-safe-map: Map with read/write transactions
- batch-reduction: Customizable reducers with batch collection
- channel-lifecycle: Built-in channel-based goroutine lifecycle management
- memfs: concurrency-safe in-memory fs.FS with os-style writes (WriteFile, MkdirAll, Remove, Rename)
- mountfs: composes named fs.FS values into one fs.FS whose top-level directories are the mounts

## Module
github.com/panyam/goutils

## Location
newstack/goutils/master

## Stack Dependencies
None

## Integration

### Go Module
```go
// go.mod
require github.com/panyam/goutils 0.1.10

// Local development
replace github.com/panyam/goutils => ~/newstack/goutils/master
```

### Key Imports
```go
import "github.com/panyam/goutils/conc"
import "github.com/panyam/goutils/memfs"
import "github.com/panyam/goutils/mountfs"
```

## Status
Stable

## Conventions
- Generic types for type safety
- Channel-based messaging
- Rob Pike concurrency patterns
