# panyam/goutils/conc

The conc package within panyam/goutils implements a few common concurrency patterns with more customizable behavior.  This was inspired by
and formalized from Rob Pike's original lecture - https://go.dev/talks/2012/concurrency.slide. 

## Examples

### Reader

A basic goroutine wrapper over a "reader" function that continuosly calls the reader function and sends read results into a channel.

### Writer

Just like the reader but for serializing writes using a writer callback method.

### Mapper

A goroutine reads from an input channel, transforms (and/or filters) it and writes the result to an output channel.

### Reducer

A goroutine collects and reduces N values from an input channel, transforms it and writes the result to an output channel.

### Pipe

A goroutine that connects a reader and a writer channel - a Mapper with the identity transform.

### Fan-In

Implementation of the fan-in pattern where multiple receiver channels can be fed into a target channel.

### Fan-Out

Implementation of the fan-out pattern where output from a single channel is fanned-out to multiple channels.

