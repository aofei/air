# Air

[![Build Status](https://travis-ci.org/sheng/air.svg?branch=master)](https://travis-ci.org/sheng/air)
[![codecov](https://codecov.io/gh/sheng/air/branch/master/graph/badge.svg)](https://codecov.io/gh/sheng/air)
[![Go Report Card](https://goreportcard.com/badge/github.com/sheng/air)](https://goreportcard.com/report/github.com/sheng/air)
[![GoDoc](https://godoc.org/github.com/sheng/air?status.svg)](https://godoc.org/github.com/sheng/air)

An ideal RESTful web framework for Go. You can use it to develop a RESTful web application as
natural as breathing.

High-performance? Fastest? Almost all the web frameworks are using these words to tell people that
they are the best. Maybe they are, maybe not. This framework does not intend to follow the crowd.
So, the Air web framework can only guarantee you one thing: **it can serve properly.**

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
	a.GET("/", homeHandler)
	a.Serve()
}

func homeHandler(c *air.Context) error {
	return c.String("Hello, 世界")
}
```

and run it

```bash
$ go run hello.go
```

then visit `http://localhost:2333`.

## Documentation

This framework is so concise that it only needs the [GoDoc](https://godoc.org/github.com/sheng/air)
enough.

## Gases

As we all know that the air is a mixture of gases. So the same is that this framework adopts the
gas as its composition. Everyone can create new gas and use it within this framework simply.

If you want to learn more about the gases, or create your own gas for this framework, simply visit
[here](https://github.com/sheng/gases).

## Examples

If you want to be familiar with this framework as soon as possible, simply visit
[here](https://github.com/sheng/atmosphere).

## Community

If you want to discuss this framework, or ask questions about it, simply post questions or ideas
[here](https://github.com/sheng/air/issues).

## Contributing

If you want to help build this framework, simply follow the
[specifications](https://github.com/sheng/air/issues/1) to send pull requests
[here](https://github.com/sheng/air/pulls).

## License

This project is licensed under the Unlicense.

License can be found [here](LICENSE).
