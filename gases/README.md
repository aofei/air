# Gases

A gas is a function chained in the HTTP request-response cycle with access to `Context` which it uses to perform a specific action, for example, logging every request or recovering from panics.

## Built-in Gases

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

