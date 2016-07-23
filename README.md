# Air

This is a Go web framework that allows people to use it as natural as breathing. You can use it to develop a dynamic website or RESTful web service without too many pains.

High-performance? Fastest? Almost all of the framework are using these words to tell people that they are the best. Maybe they are, maybe not. This framework does not intend to follow the crowd. So, the Air web framework can assure you only one sentence: **it can run properly.**

## Installation

Open your terminal and execute:

```bash
$ go get github.com/sheng/air
```

done.

> The only requirement is the [Go](https://golang.org/dl/), at least v1.6

## Hello, 世界

Create a file named `hello.go`

```go
package main

import (
	"net/http"
	"github.com/sheng/air"
)

func main() {
	a := air.New()
	a.GET("/", homeHandler)
	a.Run(":8080")
}

func homeHandler(c air.Context) error {
	return c.String(http.StatusOK, "Hello, 世界")
}
```

and run it:

```bash
$ go run hello.go
```

then open your browser and visit `http://localhost:8080`.

## Documention

* [GoDoc](https://godoc.org/github.com/sheng/air)

## Community

If you want to discuss this framework, or ask questions about it, simply post questions or ideas [here](https://github.com/sheng/air/issues).

## Contributing

If you want to help build this framework, simply send pull requests [here](https://github.com/sheng/air/pulls).

## License

This project is licensed under the TFL License.

License can be found [here](LICENSE).
