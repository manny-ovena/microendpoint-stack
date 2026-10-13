# Shared RPC transport

This package provides TCP connection management and the MessagePack request
and response envelopes shared by service RPC servers. Each connection supports
sequential requests; separate connections are served concurrently. The
service-specific handler remains responsible for method dispatch and parameter
validation.

`TCPServer` closes active connections when its context is canceled and waits
for connection handlers to finish before `Serve` returns.
