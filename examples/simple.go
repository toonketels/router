// Simple example using router.
//
// To run the app do:
//      go run examples/simple.go
//
// Go view the results by pointing your browser to
//      `http://localhost:3000/hello`
package main

import (
	"github.com/toonketels/router"
	"net/http"
)

func main() {
	// Create a new router
	appRouter := router.NewRouter()

	// Register a handlerFunc for GET/"hello" paths
	appRouter.Get("/hello", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("hello"))
	})

	// Use this router
	appRouter.Handle("/")

	// Listen for requests
	http.ListenAndServe(":3000", nil)
}
