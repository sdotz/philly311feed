package main

import (
	"net/http"

	service "github.com/sdotz/philly311"
)

func main() {
	var w http.ResponseWriter
	var req http.Request
	service.HandleProcess(w, &req)
}
