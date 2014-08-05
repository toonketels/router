// Advanced example using router.
//
// To run the app do:
//      go run examples/advanced.go
//
// Go view the results by pointing your browser to
//      `http://localhost:3000/user/20/hello`
//
// Check the logger output on the command line.
// You'll see something like
//      `GET /user/20/hello 5.916us`
package main

import (
	"fmt"
	"github.com/toonketels/router"
	"net/http"
	"time"
)

func main() {
	appRouter := router.NewRouter()

	// `Mount` mounts a handler for all paths (starting with `/`)
	// Always mount generic handlerFuncs first.
	appRouter.Mount("/", logger)

	// We can use multiple handleFuncs evaluated in order.
	// `:userid` specifies the param `userid` so it will match any string.
	appRouter.Get("/user/:userid/hello", loadUser, handleUser)

	appRouter.Handle("/")
	http.ListenAndServe(":3000", nil)
}

func logger(res http.ResponseWriter, req *http.Request) {

	// The fist handlerFunc to be executed
	// record the time when the request started
	start := time.Now()

	// Grab the current context and call
	// cntxt.Next() to handle over control to the next handlerFunc.
	// Simply dont call cntxt.Next() if you dont want to call the following
	// handlerFunc's (for instance, for access control reasons).
	router.Context(req).Next(res, req)

	// We log once all other handlerFuncs are done executing
	// so it needs to come after our call to cntxt.Next()
	fmt.Println(req.Method, req.URL.Path, time.Since(start))
}

func loadUser(res http.ResponseWriter, req *http.Request) {
	cntxt := router.Context(req)
	user, err := getUserFromDB(cntxt.Params["userid"])
	if err != nil {

		// Let the errorHandlerFunc generate the error response.
		// We stop executing the following handlers
		cntxt.Error(res, req, err.Error(), 500)
		return
	}

	// Store the value in request specific store
	_ = cntxt.Set("user", user)

	// Pass over control to next handlerFunc
	cntxt.Next(res, req)
}

func handleUser(res http.ResponseWriter, req *http.Request) {
	cntxt := router.Context(req)

	// Get a value from the request specific store
	if user, ok := cntxt.Get("user"); ok {
		if str, ok := user.(string); ok {

			// As last handlers, we should generate a response
			greeting := "Hello " + str
			res.Write([]byte(greeting))
			return
		}
	}
	res.Write([]byte("Who are you?"))

	// We dont use cntxt.Next() as there are no more
	// handlerFuncs to call. However, stuff wont explode
	// if you call cntxt.Next()` by mistake.
}

func getUserFromDB(userid string) (user string, err error) {
	user = "Richard P. F."
	err = nil
	return
}
