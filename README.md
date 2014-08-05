Router
===========================

A go (golang) package to route requests to requesthandlers.

Ideas considered (heavily borrowing from express/connect):
  - make it easy to register handlers based on HTTP verb/path combos, as this is most often used
  - complex handlerFuncs should be split up into multiple, so some can be shared between paths
  - make it easy to `mount` generic handlerFuncs which should be executed on every path
  - a path often consists of a params like userid, make it easy to register such a path and access the params by name
  - store data on a requestContext, so it can be passed to later handlerFuncs
  - set a generic errorHandlerFunc and stop executing later handerFuncs as soon as an error occurs
  - set a generic pageNotFound handlerFunc
  - use regular `http.HandlerFunc` to be compatible with existing code and go in general

Quickstart
---------------------------

After installing Go and setting up your [GOPATH](http://golang.org/doc/code.html#GOPATH), create a `server.go` file.

~~~ go
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
~~~

Then install the router package:
~~~
go get github.com/toonketels/router
~~~

Then run your server:
~~~
go run server.go
~~~


Next
-------------------------------

Take a look at the examples in the repo or run and inspect the tests.

The guide below goes a bit deeper into the api. And don't forget `godoc` for api documentation.

Guide
-------------------------------

### Table of contents
* [HandlerFuncs](#handlerfuncs)
* [Mounting handlerFuncs](#mounting-handlerfuncs)
* [Error Handling](#error-handling)


### HandlerFuncs

Once a router instance has been created, use it's `Get/Put/Post/Patch/Head/Options/Delete` methods to register a handlerFunc to a route/HTTP verb pair.

~~~ go
// Register a handlerFunc for GET/"hello" paths
appRouter.Get("/hello", handlerFunc)

// Register a handlerFunc for PUT/"hello" paths
appRouter.Put("/hello", handlerFunc)

// Register a handlerFunc for POST/"hello" paths
appRouter.Post("/hello", handlerFunc)

// Register a handlerFunc for Patch/"hello" paths
appRouter.Patch("/hello", handlerFunc)

// Register a handlerFunc for HEAD/"hello" paths
appRouter.Head("/hello", handlerFunc)

// Register a handlerFunc for OPTIONS/"hello" paths
appRouter.Options("/hello", handlerFunc)

// Register a handlerFunc for DELETE/"hello" paths
appRouter.Delete("/hello", handlerFunc)
~~~

You can also register multiple handlerFuncs for a given route.

~~~ go
// Register the handlerFuncs
appRouter.Get("/user/:userid/hello", logUser, handleUser)
~~~

For this to work, all handlerFuncs should pass control to handlerFuncs coming after them by calling
`cntxt.Next()`.

~~~ go
func loadUser(res http.ResponseWriter, req *http.Request) {
	// Grab the context for the current request
	cntxt := router.Context(req)
	// Do something

	// Pass over control to next handlerFunc
	cntxt.Next(res, req)
}
~~~

If your handlerFunc wants to protect access to certain routes, it could do so by only calling cntxt.Next() when some authorization rule validates.

~~~ go
func allowAccess(res http.ResponseWriter, req *http.Request) {
	if allowUserAccess() {
		// Allows access
		router.Context(req).Next(res, req)
		return
	}
	// Denies access
}
~~~

HandlerFuncs can store data onto the requestContext to be used by handlerFuncs after them.

~~~ go
func loadUser(res http.ResponseWriter, req *http.Request) {
	cntxt := router.Context(req)
	user := getUserFromDB(cntxt.Params["userid"])

	// Store the value in request specific store
	_ = cntxt.Set("user", user)

	cntxt.Next(res, req)
}
~~~

HandlerFuncs use cntxt.Get(key) to get the value. RequestContext has `Set/ForceSet/Get/Delete` methods all related to the data stored during the current request.

~~~ go
func handleUser(res http.ResponseWriter, req *http.Request) {
	cntxt := router.Context(req)

	// Get a value from the request specific store
	if user, ok := cntxt.Get("user"); ok {
		// Do something
	}
	// Do something else
}
~~~

Remember the route `/user/:userid/hello`? It matches routes like `/user/14/hello` or `/user/richard/hello`. HandlerFuncs can access the values of `userid` on the requestContext.

~~~ go
func loadUser(res http.ResponseWriter, req *http.Request) {

	// Grab the userid param from the context
	userid := router.Context(req).Params["userid"]
	
	// Do something with it
}
~~~

As you might have noticed, handlers need to be http.HandlerFunc's. So you can use your existing ones if you don't need to access the requestContext.


### Mounting handlerFuncs

Some handlerFuncs need to be executed for every request, like a logger. Instead of passing it to every registered route, we "mount" them using `router.Mount()`.

~~~ go
appRouter := router.NewRouter()

// Mount handlerFuncs first
appRouter.Mount("/", logger)
appRouter.Mount("/", allowAccess)

// Then start matching paths
appRouter.Get("/user/:userid/hello", loadUser, handleUser)
~~~

The order in which you mounted and registered handlers, is the order in which they will be executed.

A request to `/user/14/hello` will result in `logger` to be called first, followed by `allowAccess`, `loadUser` and `handleUser`. That is as long as none of the handlerFunc's prevented the latter ones from executing by not calling next.

By changing the mountPath of allowAccess to `/admin`, we get different results.

~~~ go
// Mount handlerFuncs first
appRouter.Mount("/", logger)
appRouter.Mount("/admin", allowAccess)

// Then start matching paths
appRouter.Get("/user/:userid/hello", loadUser, handleUser)
appRouter.Get("/admin/user/:userid", loadUser, administerUserHandler)
~~~

It the above case a request to `/user/20/hello` will execute `logger -> loadUser -> handleUser`, while a request to `/admin/user/20` executes `logger -> allowAccess -> loadUser -> administerUserHandler`.

We see that by dividing a complex handlerFunc into multiple smaller ones, we get more code reuse. It becomes easy to create a small set of "middleware" handlers to be reused on different routes. While the last handlerFunc is generally the one responsible for generating the actual response.

### Error Handling

Besides storing data and dispatching the next handlerFunc cntxt has an `Error` method. Let's update the `loadUser` handlerFunc to take errors into account.

~~~ go
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
~~~

Calling `cntxt.Error()` notifies the requestContext an error has been made and further `Next()` call will be prevented. It delegates the requestHandling to a dedicated `errorHandlerFunc` to reply in a consistent manner. 

Though calling `Next()` after an error will never dispatch the next HandlerFunc, it is wise to just return after the error so the current
HandlerFunc stops executing.

Previous HandlerFuncs are allowed to continue executing their code when the executing flow returns to them.

For instance, when the logger below is called, it records the time and calls the next HandlerFunc. If that handler errs, logger will resume as usual allowing it to log its output. 

~~~ go
func logger(res http.ResponseWriter, req *http.Request) {

	// The fist handlerFunc to be executed
	// record the time when the request started
	start := time.Now()

	// Handle over control to the next handlerFunc.
	router.Context(req).Next(res, req)

	// We log once all other handlerFuncs are done executing
	// so it needs to come after our call to cntxt.Next()
	fmt.Println(req.Method, req.URL.Path, time.Now().Sub(start))
}
~~~ 

The actual response send to the client is handled by default ErrorRequestHandler. Which will just do `http.Error(res, err, code)`.

Customise the response by updating your router's `ErrorHandler`.

~~~ go
appRouter.ErrorHandler = func(res http.ResponseWriter, req *http.Request, err string, code int) {
	http.Error(res, strings.ToUpper(err), code)
}
~~~

The request is passed to allow a different response to be send depending on request properties.

Similarly, configure the response generated when a route is not found by updating the router's `NotFoundHandler` which is a plain http.HandlerFunc.