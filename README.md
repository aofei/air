# Air

[![GitHub Actions](https://github.com/aofei/air/workflows/Main/badge.svg)](https://github.com/aofei/air)
[![codecov](https://codecov.io/gh/aofei/air/branch/master/graph/badge.svg)](https://codecov.io/gh/aofei/air)
[![Go Report Card](https://goreportcard.com/badge/github.com/aofei/air)](https://goreportcard.com/report/github.com/aofei/air)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/aofei/air)](https://pkg.go.dev/github.com/aofei/air)

An ideally refined web framework for Go.

High-performance? Fastest? Almost all web frameworks are using these words to
tell people that they are the best. Maybe they are, maybe not. Air does not
intend to follow the crowd. Our goal is always to strive to make it easy for
people to use Air to build their web applications. So, we can only guarantee you
one thing: **Air can serve properly**.

## Features

* API
	* As less as possible
	* As clean as possible
	* As simple as possible
	* As expressive as possible
* Server
	* HTTP/2 (h2 & h2c) support
	* SSL/TLS support
	* ACME support
	* PROXY (v1 & v2) support
	* Graceful shutdown support
* Router
	* Based on the Radix Tree
	* Zero dynamic memory allocation
	* Blazing fast
	* Has a good inspection mechanism
	* Group routes support
* Gas (aka middleware)
	* Router level:
		* Before router
		* After router
	* Route level
	* Group level
* WebSocket
	* Full-duplex communication
* Reverse proxy
	* Retrieves resources on behalf of a client from another server
	* Supported protocols:
		* HTTP
		* WebSocket
		* gRPC
* Binder
	* Binds HTTP request body into the provided struct
	* Supported MIME types:
		* `application/json`
		* `application/xml`
		* `application/protobuf`
		* `application/msgpack`
		* `application/toml`
		* `application/yaml`
		* `application/x-www-form-urlencoded`
		* `multipart/form-data`
* Renderer
	* Rich template functions
	* Hot update support
* Minifier
	* Minifies HTTP response on the fly
	* Supported MIME types:
		* `text/html`
		* `text/css`
		* `application/javascript`
		* `application/json`
		* `application/xml`
		* `image/svg+xml`
* Gzip
	* Compresses HTTP response by using the gzip
	* Default MIME types:
		* `text/plain`
		* `text/html`
		* `text/css`
		* `application/javascript`
		* `application/json`
		* `application/xml`
		* `application/toml`
		* `application/yaml`
		* `image/svg+xml`
* Coffer
	* Accesses binary asset files by using the runtime memory
	* Significantly improves the performance of the [`air.Response.WriteFile`](https://pkg.go.dev/github.com/aofei/air#Response.WriteFile)
	* Asset file minimization and gzip support
	* Default asset file extensions:
		* `.html`
		* `.css`
		* `.js`
		* `.json`
		* `.xml`
		* `.toml`
		* `.yaml`
		* `.yml`
		* `.svg`
		* `.jpg`
		* `.jpeg`
		* `.png`
		* `.gif`
	* Hot update support
* I18n
	* Adapt to the request's favorite conventions
	* Implanted into the [`air.Response.Render`](https://pkg.go.dev/github.com/aofei/air#Response.Render)
	* Hot update support
* Error
	* Centralized handling

## Installation

Open your terminal and execute

```bash
$ go get github.com/aofei/air
```

done.

> The only requirement is the [Go](https://golang.org), at least v1.13.

## Hello, 世界

Create a file named `hello.go`

```go
package main

import "github.com/aofei/air"

func main() {
	air.Default.GET("/", func(req *air.Request, res *air.Response) error {
		return res.WriteString("Hello, 世界")
	})
	air.Default.Serve()
}
```

and run it

```bash
$ go run hello.go
```

then visit `http://localhost:8080`.

## Documentation

Does all web frameworks need to have a complicated (or a lovely but lengthy)
website to guide people how to use them? Well, Air has only one
[Doc](https://pkg.go.dev/github.com/aofei/air) with useful comments. In fact,
Air is so succinct that you don't need to understand how to use it through a
large document.

## Gases

As we all know that the air of Earth is a mixture of gases. So the same is that
Air adopts the gas as its composition. Everyone can create new gas and use it
within Air simply.

A gas is a function chained in the HTTP request-response cycle with access to
the [`air.Request`](https://pkg.go.dev/github.com/aofei/air#Request) and
[`air.Response`](https://pkg.go.dev/github.com/aofei/air#Response) which it uses
to perform a specific action, for example, logging every request or recovering
from panics.

```go
return func(next air.Handler) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		// Do something here...
		return next(req, res) // Execute the next handler
	}
}
```

If you already have some good HTTP middleware, you can simply wrap them into
gases by calling the
[`air.WrapHTTPMiddleware`](https://pkg.go.dev/github.com/aofei/air#WrapHTTPMiddleware).

If you are looking for some useful gases, simply visit
[here](https://github.com/air-gases).

## Examples

If you want to be familiar with Air as soon as possible, simply visit
[here](https://github.com/air-examples).

## FAQ

### Why named Air?

"A" for "An", "I" for "Ideally" and "R" for "Refined". So, Air.

### Why based on the [net/http](https://pkg.go.dev/net/http)?

In fact, I've tried to implement a full-featured HTTP server (just like the
awesome [valyala/fasthttp](https://github.com/valyala/fasthttp)). But when I
finished about half of the work, I suddenly realized: What about stability? What
about those awesome middleware outside? And, seriously, what am I doing?

### Why not just use the [net/http](https://pkg.go.dev/net/http)?

Yeah, we can of course use the [net/http](https://pkg.go.dev/net/http) directly,
after all, it can meet many requirements. But, ummm... it's really too stable,
isn't it? I mean, to ensure Go's backward compatibility (which is extremely
necessary), we can't easily add some handy features to the
[net/http](https://pkg.go.dev/net/http). And, the
[`http.Request`](https://pkg.go.dev/net/http#Request) does not only represents
the request received by the server, but also represents the request made by the
client. In some cases it can be confusing. So why not just use the
[net/http](https://pkg.go.dev/net/http) as the underlying server, and then
implement a refined web framework that are only used for the server-side on top
of it?

### Do you know we already got the [gin-gonic/gin](https://github.com/gin-gonic/gin) and [labstack/echo](https://github.com/labstack/echo)?

Of course, I knew it when I started Go. And, I love both of them! But, why not
try some new flavors? Are you sure you prefer them instead of Air? Don't even
give Air a try? Wow... well, maybe Air is not for you. After all, it's for
people who love to try new things. Relax and continue to maintain the status
quo, you will be fine.

### What about the fantastic [Gorilla web toolkit](https://github.com/gorilla)?

Just call the
[`air.WrapHTTPHandler`](https://pkg.go.dev/github.com/aofei/air#WrapHTTPHandler)
and
[`air.WrapHTTPMiddleware`](https://pkg.go.dev/github.com/aofei/air#WrapHTTPMiddleware).

### Is Air good enough?

Far from enough. But it's already working.

## Community

If you want to discuss Air, or ask questions about it, simply post questions or
ideas [here](https://github.com/aofei/air/issues).

## Contributing

If you want to help build Air, simply follow
[this](https://github.com/aofei/air/wiki/Contributing) to send pull requests
[here](https://github.com/aofei/air/pulls).

## License

This project is licensed under the Unlicense.

License can be found [here](LICENSE).
