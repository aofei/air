# Air

[![Build Status](https://travis-ci.org/aofei/air.svg?branch=master)](https://travis-ci.org/aofei/air)
[![Coverage Status](https://coveralls.io/repos/github/aofei/air/badge.svg?branch=master)](https://coveralls.io/github/aofei/air?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/aofei/air)](https://goreportcard.com/report/github.com/aofei/air)
[![GoDoc](https://godoc.org/github.com/aofei/air?status.svg)](https://godoc.org/github.com/aofei/air)

An ideally refined web framework for Go. You can use it to build a web
application as natural as breathing.

High-performance? Fastest? Almost all the web frameworks are using these words
to tell people that they are the best. Maybe they are, maybe not. Air does not
intend to follow the crowd. It can only guarantee you one thing: **it can serve
properly.**

## FAQ

**Q: Why named Air?**

A: "A" for "An", "I" for "Ideally" and "R" for "Refined". So, Air.

**Q: Why based on the `net/http`?**

A: In fact, I've tried to implement a full-featured HTTP server (just like the
awesome [valyala/fasthttp](https://github.com/valyala/fasthttp)). But when I
finished about half of the work, I suddenly realized: What about stability? What
about those awesome middleware outside? And, seriously, what am I doing?

**Q: Why not just use the `net/http`?**

A: Yeah, we can of course use the `net/http` directly, after all, it can meet
many requirements. But, ummm... It's really too stable, isn't it? I mean, to
ensure Go's backward compatibility (which is extremely necessary), we can't
easily add some handy features to the `net/http`. And, the `http.Request` does
not only represents the request received by the server, but also represents the
request made by the client. In some cases it can be confusing. So why not just
use the `net/http` as the underlying server, and then implement a refined web
framework that are only used for the server-side on top of it?

**Q: Do you know we already got the
[gin-gonic/gin](https://github.com/gin-gonic/gin) and the
[labstack/echo](https://github.com/labstack/echo)?**

A: Of course, I knew it when I started Go. And, I love both of them! But, why
not try some new flavors? Are you sure you prefer them instead of Air? Don't
even give Air a try? Wow... Well, maybe Air is not for you. After all, it's for
people who love to try new things. Relax and continue to maintain the status
quo, you will be fine.

**Q: What about the fantastic
[Gorilla web toolkit](https://github.com/gorilla)?**

A: Just call the `air.WrapHTTPMiddleware()`.

**Q: Is Air good enough?**

A: Far from enough. But it's already working.

## Features

* API
	* As less as possible
	* As simple as possible
	* As expressive as possible
* Method
	* `GET` (cache-friendly)
	* `HEAD` (cache-friendly)
	* `POST`
	* `PUT`
	* `PATCH`
	* `DELETE`
	* `CONNECT`
	* `OPTIONS`
	* `TRACE`
* Logger
	* `DEBUG`
	* `INFO`
	* `WARN`
	* `ERROR`
	* `FATAL`
	* `PANIC`
* Server
	* HTTP/2 support
	* SSL/TLS support
	* ACME support
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
	* Protocols:
		* HTTP(S)
		* WebSocket
		* gRPC
* Binder
	* `application/json`
	* `application/xml`
	* `application/msgpack`
	* `application/x-msgpack`
	* `application/protobuf`
	* `application/x-protobuf`
	* `application/toml`
	* `application/x-toml`
	* `application/x-www-form-urlencoded`
	* `multipart/form-data`
* Minifier
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
		* `image/svg+xml`
* Renderer
	* Rich template functions
	* Hot update support
* Coffer
	* Accesses binary asset files by using the runtime memory
	* Significantly improves the performance of the `air.Response#File()`
	* Asset file minimization:
		* `.html`
		* `.css`
		* `.js`
		* `.json`
		* `.xml`
		* `.svg`
		* `.jpg`
		* `.jpeg`
		* `.png`
	* Hot update support
* I18n
	* Adapt to the request's favorite conventions
	* Implanted into the `air.Response#Render()`
	* Hot update support
* Error
	* Centralized handling

## Installation

Open your terminal and execute

```bash
$ go get github.com/aofei/air
```

done.

> The only requirement is the [Go](https://golang.org), at least v1.9.

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

Does all the web frameworks need to have a complicated (or a lovely but lengthy)
website to guide people how to use them? Well, Air has only one
[GoDoc](https://godoc.org/github.com/aofei/air) with useful comments. In fact,
Air is so succinct that you don't need to understand how to use it through a
large document.

## Gases

As we all know that the air is a mixture of gases. So the same is that this
framework adopts the gas as its composition. Everyone can create new gas and use
it within this framework simply.

A gas is a function chained in the HTTP request-response cycle with access to
the `air.Request` and the `air.Response` which it uses to perform a specific
action, for example, logging every request or recovering from panics.

If you have got some good HTTP middleware, you can simply wrap them into gases
by calling the `air.WrapHTTPMiddleware()`.

If you are looking for some useful gases, simply visit
[here](https://github.com/air-gases).

## Examples

If you want to be familiar with this framework as soon as possible, simply visit
[here](https://github.com/air-examples).

## Community

If you want to discuss this framework, or ask questions about it, simply post
questions or ideas [here](https://github.com/aofei/air/issues).

## Contributing

If you want to help build this framework, simply follow
[this](https://github.com/aofei/air/wiki/Contributing) to send pull requests
[here](https://github.com/aofei/air/pulls).

## License

This project is licensed under the Unlicense.

License can be found [here](LICENSE).
