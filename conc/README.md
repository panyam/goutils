# panyam/goutils/conc

The conc package within panyam/goutils implements a few common concurrency patterns with more customizable behavior.  This was inspired by
and formalized from Rob Pike's original lecture - https://go.dev/talks/2012/concurrency.slide. 

## Examples

### Reader

A basic go-routine wrapper over a "reader" function.  The idea is that the callback will continuously call a reader method (eg reading json messages from a network stream) and passes onto a channel.

### Writer

Just like the reader but for serializing writes onto a writer callback method.

### Pipe

A go-routine that connects a reader and a writer channel.

### Fan-In

Implementation of the fan-in pattern where multiple receiver channels can be selected to feed into a target channel.

### Fan-Out

Implementation of the fan-out pattern where output from a single channel is fanned-out to multiple channels.

