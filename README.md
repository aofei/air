# Air

[![Build Status](https://travis-ci.org/sheng/air.svg?branch=master)](https://travis-ci.org/sheng/air)
[![codecov](https://codecov.io/gh/sheng/air/branch/master/graph/badge.svg)](https://codecov.io/gh/sheng/air)
[![Go Report Card](https://goreportcard.com/badge/github.com/sheng/air)](https://goreportcard.com/report/github.com/sheng/air)
[![GoDoc](https://godoc.org/github.com/sheng/air?status.svg)](https://godoc.org/github.com/sheng/air)

An ideal RESTful web framework for Go. You can use it to develop a RESTful web
application as natural as breathing.

High-performance? Fastest? Almost all the web frameworks are using these words
to tell people that they are the best. Maybe they are, maybe not. This framework
does not intend to follow the crowd. So, Air web framework can only guarantee
you one thing: **it can serve properly.**

## Features

* APIs
	* As less as possible.
	* As simple as possible.
	* As expressive as possible.
* HTTP Methods
	* `GET`
	* `HEAD`
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
	* `PANIC`
	* `FATAL`
	* Powered by the Go `text/template`.
* Server
	* HTTP/2 support.
	* SSL/TLS support.
	* Gracefully shutdown support.
	* Powered by the Go `net/http`.
* Router
	* Based on the Radix Tree.
	* Has a good inspection mechanism.
	* Group routes support.
* Gas (also called middleware)
	* Router level:
		* Before router.
		* After router.
	* Route level.
	* Group level.
* Binder
	* Based on the `Content-Type` header.
	* Supported MIME types:
		* `application/json`.
		* `application/xml`.
		* `application/x-www-form-urlencoded`.
* Minifier
	* Supported MIME types:
		* `text/html`
		* `text/css`
		* `text/javascript`
		* `application/json`
		* `text/xml`
		* `image/svg+xml`
		* `image/jpeg`
		* `image/png`
	* Powered by the Go `image` and the [minify](https://github.com/tdewolff/minify).
* Renderer
	* Rich template functions.
	* Hot update support by using the [fsnotify](https://github.com/fsnotify/fsnotify).
	* Powered by the Go `html/template`.
* Coffer
	* Accesses binary asset files by using the runtime memory.
	* Reduces the hard disk I/O and significantly improves the performance of the `Response#File()`.
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
	* Hot update support by using the [fsnotify](https://github.com/fsnotify/fsnotify).
* Error
	* Centralized handling.

## Installation

Open your terminal and execute

```bash
$ go get github.com/sheng/air
```

done.

> The only requirement is the [Go](https://golang.org/dl/), at least v1.8.

## Hello, 世界

Create a file named `hello.go`

```go
package main

import "github.com/sheng/air"

func main() {
	a := air.New()
	a.GET("/", func(req *air.Request, res *air.Response) error {
		return res.String("Hello, 世界")
	})
	a.Serve()
}
```

and run it

```bash
$ go run hello.go
```

then visit `http://localhost:2333`.

## Documentation

* [English](https://github.com/sheng/air/wiki/Documentation)
* [简体中文](https://github.com/sheng/air/wiki/文档)
* [GoDoc](https://godoc.org/github.com/sheng/air)

## Gases

As we all know that the air is a mixture of gases. So the same is that this
framework adopts the gas as its composition. Everyone can create new gas and use
it within this framework simply.

A gas is a function chained in the HTTP request-response cycle with access to
`air.Request` and `air.Response` which it uses to perform a specific action, for
example, logging every request or recovering from panics.

## Examples

If you want to be familiar with this framework as soon as possible, simply visit
[here](https://github.com/sheng/atmosphere).

## Community

If you want to discuss this framework, or ask questions about it, simply post
questions or ideas [here](https://github.com/sheng/air/issues).

## Contributing

If you want to help build this framework, simply follow
[these](https://github.com/sheng/air/wiki/Contributing) to send pull requests
[here](https://github.com/sheng/air/pulls).

## License

This project is licensed under the Unlicense.

License can be found [here](LICENSE).
