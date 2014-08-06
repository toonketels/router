/*
Package router provides a simple yet powerful URL router and HandlerFunc dispatcher for web apps.

Ideas considered (heavily borrowing from express/connect):
  - registering handlers based on HTTP verb/path combos should be easy, as this is most often used
  - split complex HandlerFuncs into multiple smaller one which can be shared
  - mount generic HandlerFuncs to be executed on every path
  - registering and accessing paths with params (like :userid) should be easy
  - store data on a requestContext, so it can be passed to later HandlerFuncs
  - set a generic errorHandlerFunc and stop executing later handerFuncs as soon as an error occurs
  - set a generic pageNotFound HandlerFunc
  - handlers are regular `http.HandlerFunc` to be compatible with go

Basic usage

	// Create a new router
	appRouter := router.NewRouter()

	// Register a HandlerFunc for GET/"hello" paths
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

		// `Mount` mounts a handler for all paths (starting with `/`)
		// Always mount generic HandlerFuncs first.
		appRouter.Mount("/", logger)

		// We can use multiple handleFuncs evaluated in order.
		// `:userid` specifies the param `userid` so it will match any string.
		appRouter.Get("/user/:userid/hello", loadUser, handleUser)

		appRouter.Handle("/")
		http.ListenAndServe(":3000", nil)
	}

	func logger(res http.ResponseWriter, req *http.Request) {

		// The fist HandlerFunc to be executed
		// record the time when the request started
		start := time.Now()

		// Grab the current context and call
		// cntxt.Next() to handle over control to the next HandlerFunc.
		// Simply don't call cntxt.Next() if you don't want to call the following
		// HandlerFunc's (for instance, for access control reasons).
		router.Context(req).Next(res, req)

		// We log once all other HandlerFuncs are done executing
		// so it needs to come after our call to cntxt.Next()
		fmt.Println(req.Method, req.URL.Path, time.Since(start))
	}

	func loadUser(res http.ResponseWriter, req *http.Request) {
		cntxt := router.Context(req)
		user, err := getUserFromDB(cntxt.Params["userid"])
		if err != nil {

			// Let the ErrorHandler generate the error response.
			// We stop executing the following handlers
			cntxt.Error(res, req, err.Error(), 500)
			return
		}

		// Store the value in request specific store
		_ = cntxt.Set("user", user)

		// Pass over control to next HandlerFunc
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
		// HandlerFuncs to call. However, stuff wont explode
		// if you call cntxt.Next()` by mistake.
	}

	// func getUserFromDB...

This will log for each request. On "/user/:userid/hello" matching paths, it loads a user and saves it to the requestContext store and handleUser generates the response.
Note all handlers are regular http.HandlerFunc and use a `Context` to hand over control and data to the next HandlerFunc.
*/
package router
