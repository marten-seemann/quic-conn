# A QUIC Connection

At the moment, this project is intended to figure out the right API exposed by the [quic package in quic-go](https://github.com/lucas-clemente/quic-go).

When fully implemented, a QUIC connection can be used as a replacement for an encrypted TCP connection. It provides a single ordered byte-stream abstraction, with the main benefit of being able to perform connection migration.

## Usage of the example

Start listening for an incoming QUIC connection
```go
go run example/main.go -s
```
The server will echo every message received on the connection in uppercase.

Send a message on the QUIC connection:
```go
go run example/main.go -c
```
