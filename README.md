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

## Features

* Singleton
	* Air is uncountable
	* Just one package `air.*`
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
	* WebSocket support
* Router
	* Based on the Radix Tree
	* Has a good inspection mechanism
	* Group routes support
* Gas (also called middleware)
	* Router level:
		* Before router
		* After router
	* Route level
	* Group level
* Binder
	* `application/json`
	* `application/xml`
	* `application/x-www-form-urlencoded`
	* `multipart/form-data`
* Minifier
	* `text/html`
	* `text/css`
	* `application/javascript`
	* `application/json`
	* `application/xml`
	* `image/svg+xml`
	* `image/jpeg`
	* `image/png`
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
* Error
	* Centralized handling

## Installation

Open your terminal and execute

```bash
$ go get github.com/aofei/air
```

done.

> The only requirement is the [Go](https://golang.org), at least v1.8.

## Hello, 世界

Create a file named `hello.go`

```go
package main

import "github.com/aofei/air"

func main() {
	air.GET("/", func(req *air.Request, res *air.Response) error {
		return res.WriteString("Hello, 世界")
	})
	air.Serve()
}
```

and run it

```bash
$ go run hello.go
```

then visit `http://localhost:2333`.

## Documentation

* [English](https://github.com/aofei/air/wiki/Documentation)
* [简体中文](https://github.com/aofei/air/wiki/文档)
* [GoDoc](https://godoc.org/github.com/aofei/air)

## Gases

As we all know that the air is a mixture of gases. So the same is that this
framework adopts the gas as its composition. Everyone can create new gas and use
it within this framework simply.

A gas is a function chained in the HTTP request-response cycle with access to
`air.Request` and `air.Response` which it uses to perform a specific action, for
example, logging every request or recovering from panics.

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
