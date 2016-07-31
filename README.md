# Air

This is a Go web framework that allows people to use it as natural as breathing. You can use it to develop a dynamic website or RESTful web service without too many pains.

High-performance? Fastest? Almost all of the framework are using these words to tell people that they are the best. Maybe they are, maybe not. This framework does not intend to follow the crowd. So, the Air web framework can assure you only one sentence: **it can run properly.**

## Installation

Open your terminal and execute

```bash
$ go get github.com/sheng/air
```

done.

> The only requirement is the [Go](https://golang.org/dl/), at least v1.5

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
	a.Get("/", homeHandler)
	a.Run(":8080")
}

func homeHandler(c *air.Context) error {
	return c.String(http.StatusOK, "Hello, 世界")
}
```

and run it

```bash
$ go run hello.go
```

then open your browser and visit `http://localhost:8080`.

## Documention

* English(editing...)
* 简体中文(编辑中...)
* [GoDoc](https://godoc.org/github.com/sheng/air)

## Gas

As we all know that the Air is a mixture of a combination of multiple gases. So the same is that this framework adopts the Gas as its composition. Everyone can create new Gas and use it within this framework.

**Built-in Gases**

Gas | Description
--- | ---
[Logger](https://godoc.org/github.com/sheng/air/gases#Logger) | Log HTTP requests
[Recover](https://godoc.org/github.com/sheng/air/gases#Recover) | Recover from panics
[Gzip](https://godoc.org/github.com/sheng/air/gases#Gzip) | Send gzip HTTP response
[JWT](https://godoc.org/github.com/sheng/air/gases#JWT) | JWT authentication
[Secure](https://godoc.org/github.com/sheng/air/gases#Secure) | Protection against attacks
[CORS](https://godoc.org/github.com/sheng/air/gases#CORS) | Cross-Origin Resource Sharing
[CSRF](https://godoc.org/github.com/sheng/air/gases#CSRF) | Cross-Site Request Forgery
[Static](https://godoc.org/github.com/sheng/air/gases#Static) | Serve static files

**Add-on Gases**

If you want to find other Gases, or create your own Gas for this framework, simply visit [here](https://github.com/air-gases).

## Community

If you want to discuss this framework, or ask questions about it, simply post questions or ideas [here](https://github.com/sheng/air/issues).

## Contributing

If you want to help build this framework, simply send pull requests [here](https://github.com/sheng/air/pulls).

## License

This project is licensed under the TFL License.

License can be found [here](LICENSE).
