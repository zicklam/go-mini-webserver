# go-mini-webserver

Run a simple HTTP server to serve 1 html page with access_log.

## Download

`go get github.com/zicklam/go-mini-webserver`

## Usage
```
Usage of ./go-mini-webserver:
  -document string
        Wat soll ich zeigen? (default "index.html")
  -listen string
        Worauf soll ich hören? (default ":8080")
```

## Example

`./go-mini-webserver -listen :8080 -document /var/www/html/index.html`

# License

[![License: Unlicense](https://img.shields.io/badge/license-Unlicense-blue.svg)](http://unlicense.org/)
