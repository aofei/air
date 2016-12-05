# Air

An ideal RESTful web framework for Go. You can use it to develop a dynamic website or RESTful web
service as natural as breathing.

High-performance? Fastest? Almost all of the web frameworks are using these words to tell people
that they are the best. Maybe they are, maybe not. This framework does not intend to follow the
crowd. So, the Air web framework can assure you only one sentence: **it can run properly.**

## Installation

Open your terminal and execute

```bash
$ go get github.com/sheng/air
```

done.

> The only requirement is the [Go](https://golang.org/dl/), at least v1.7

## Hello, 世界

Create a file named `hello.go`

```go
package main

import "github.com/sheng/air"

func main() {
	a := air.New()
	a.GET("/", homeHandler)
	a.Run()
}

func homeHandler(c *air.Context) error {
	c.Data["string"] = "Hello, 世界"
	return c.String()
}
```

and run it

```bash
$ go run hello.go
```

then open your browser and visit `http://localhost:8080`.

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

This project is licensed under the TFL License.

License can be found [here](LICENSE).
