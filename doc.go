/*
Router provides a simple yet powerful URL router and handlerFunc dispatcher for web apps.

Ideas considered (heavily borrowing from express/connect):
  - make it easy to register handlers based on HTTP verb/path combos, as this is most often used
  - complex handlerFuncs should be split up into multiple, so some can be shared between paths
  - make it easy to `mount` generic handlerFuncs which should be executed on every path
  - a path often consists of a params like userid, make it easy to register such a path and access the params by name
  - store data on a requestContext, so it can be passed to later handlerFuncs
  - set a generic errorHandlerFunc and stop executing later handerFuncs as soon as an error occurs
  - set a generic pageNotFound handlerFunc
  - use regular `http.HandlerFunc` to be compatible with existing code and go in general

Basic usage

	// Create a new router
	appRouter = router.newRouter()

	// Register a handlerFunc for GET/"hello" paths
	appRouter.Get("/hello", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("hello"))
	})

	// Use this router
	appRouter.Handle("/")

	// Listen for requests
	http.ListenAndServe(":3000", nil)

More advanced usage

	func main() {
		appRouter := router.NewRouter()

		// `Use` mounts a handler for all paths (starting with `/`)
		// Always mount generic handlerFuncs first.
		appRouter.Use("/", logger)

		// We can use multiple handleFuncs evaluated in order
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
		fmt.Println(req.Method, req.URL.Path, time.Now().Sub(start))
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

	// func getUserFromDB...

This will ensure a logger is used on each request, a user loaded on saved to requestContext store, and the response handled by handleUser.
Note all handlers are regular http.HandlerFunc and use a `Context` to hand over control and data to the next handlerFunc.
*/
package router
