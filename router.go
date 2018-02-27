package router

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// Stores
// ----------------------

// Store to keep track of the current requestContexts in use.
var requestContextStore sync.Map // map[*http.Request]*RequestContext

// Router
// ----------------------

// A Router to register paths and requestHandlers to.
//
// Set a custom NotFoundHandler if you want to override go's default one.
//
// There can be multiple per application, if so, don't forget to pass a
// different pattern to `router.Handle()`.
type Router struct {
	NotFoundHandler http.HandlerFunc // Specify a custom NotFoundHandler
	ErrorHandler    ErrorHandler     // Specify a custom ErrorHandler
	routes          map[string][]*requestHandler
	mounted         []mountedRequestHandler
}

// NewRouter creates a router and returns a pointer to it so
// you can start registering routes.
//
// Don't forget to call `router.Handle(pattern)` to actually use
// the router.
func NewRouter() (router *Router) {
	router = new(Router)

	router.routes = map[string][]*requestHandler{
		"GET":     make([]*requestHandler, 0),
		"POST":    make([]*requestHandler, 0),
		"PUT":     make([]*requestHandler, 0),
		"DELETE":  make([]*requestHandler, 0),
		"PATCH":   make([]*requestHandler, 0),
		"OPTIONS": make([]*requestHandler, 0),
		"HEAD":    make([]*requestHandler, 0),
	}

	// Ensure we have an error handler set
	router.ErrorHandler = defaultErrorHandler
	return
}

// Get registers a GET path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Get(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("GET", path, handlers...)
}

// Post registers a POST path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Post(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("POST", path, handlers...)
}

// Put registers a PUT path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Put(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("PUT", path, handlers...)
}

// Delete registers a DELETE path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Delete(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("DELETE", path, handlers...)
}

// Patch registers a PATCH path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Patch(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("PATCH", path, handlers...)
}

// Options registers an OPTONS path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Options(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("OPTIONS", path, handlers...)
}

// Head registers an HEAD path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted HandlerFuncs).
func (router *Router) Head(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("HEAD", path, handlers...)
}

// Mount mounts a requestHandler for a given mountPath. The requestHandler
// will be executed on all paths which start like the mountPath.
//
// For example: mountPath "/" will execute the requestHandler for all requests
// (each one starts with "/"), contrary to "/api" will only execute the
// handler on paths starting with "/api" like "/api", "/api/2", "api/users/23"...
//
// This allows for the use of general middleware, unlike more specific middleware
// handlers which are registered on specific paths.
//
// Use mount if your requestHandlers needs to be invoked on all/most paths so you
// don't have to register it again and again when registering handlers.
//
// The mountPath don't accept tokens (like :user) but can access the params on
// the context if the path on which it is fired contains those tokens.
func (router *Router) Mount(mountPath string, handler http.HandlerFunc) {
	mReqHandler := mountedRequestHandler{
		MountPath: mountPath,
		Handle:    handler,
		Matcher:   regexp.MustCompile(`^\` + mountPath),
	}
	router.mounted = append(router.mounted, mReqHandler)
}

// Handle registers the router for the given pattern in the DefaultServeMux.
// The documentation for ServeMux explains how patterns are matched.
//
// This delegates to `http.Handle()` internally.
//
// Most of the times, you just want to do `router.Handle("/")`.
func (router *Router) Handle(pattern string) {
	http.Handle(pattern, router)
}

// Needed by go to actually start handling the registered routes.
// You don't need to call this yourself.
func (router *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	unMatched := true

	// For each of the registered routes for this request method...
	for _, reqHandler := range router.routes[req.Method] {
		// Only when the route matches...
		if isAMatch, withParams := reqHandler.matches(req.URL.Path); isAMatch {
			unMatched = false

			// Create a RequestContext
			cntxt := new(RequestContext)
			// Store the requestContext
			requestContextStore.Store(req, cntxt)
			// Capture the route params
			cntxt.Params = withParams
			// Attach the handlers to the context
			cntxt.handlers = reqHandler.Handlers
			// Set the ErrorHandler
			cntxt.errorHandler = router.ErrorHandler
			// Dispatch the first handler,
			// the request is being served.
			cntxt.Next(res, req)
			// Clean up
			requestContextStore.Delete(req)
			break
		}
	}

	// Nothing found...
	if unMatched {
		router.notFound(res, req)
	}
}

// Helper function to actually register the requestHandler on the router.
func (router *Router) registerRequestHandler(method string, path string, handlers ...http.HandlerFunc) {
	reqHandler := router.makeRequestHandler(path, handlers...)
	router.routes[method] = append(router.routes[method], reqHandler)
}

// Helper function to dispatch the correct NotFoundHandler.
func (router *Router) notFound(res http.ResponseWriter, req *http.Request) {
	if router.NotFoundHandler != nil {
		router.NotFoundHandler(res, req)
	} else {
		http.NotFound(res, req)
	}
}

// Creates the requestHandler struct from the given path
func (router *Router) makeRequestHandler(path string, handlers ...http.HandlerFunc) (reqHandler *requestHandler) {
	// Mount middleware
	handlersToMount := router.handlersToMountFor(path)
	// Make the mountedMiddleware the first handlers to be called
	// followed by our registered handlers... keeping everything in order
	handlers = append(handlersToMount, handlers...)

	// Build the regexp string to match each incoming request against
	regexpPath, withParamNames := buildRegexpFor(path)

	reqHandler = &requestHandler{
		Path:       path,
		ParamNames: withParamNames,
		Regex:      regexp.MustCompile(regexpPath),
		Tokenized:  len(withParamNames) != 0,
		Handlers:   handlers,
	}
	return
}

// Returns all mountedRequestHandlers that should be mounted for the given path.
func (router *Router) handlersToMountFor(path string) (mountedMiddleware []http.HandlerFunc) {
	mountedMiddleware = make([]http.HandlerFunc, 0)
	for _, mReqHandler := range router.mounted {
		if mReqHandler.shouldMount(path) {
			mountedMiddleware = append(mountedMiddleware, mReqHandler.Handle)
		}
	}
	return
}

// Private helper funcs
// ---------------------------

// Some paths use tokens like "/user/:userid" where "userid" is the token.
//
// This function builds a string to be compiled as a regexp to match those
// paths and returns the names of the parameters found in the route.
func buildRegexpFor(path string) (regexpPath string, withParamNames []string) {
	var items []string
	parts := strings.Split(path, "/")
	withParamNames = make([]string, 0)
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			nameOnly := strings.Trim(part, ":")
			withParamNames = append(withParamNames, nameOnly)
			items = append(items, `([^\/]+)`)
		} else {
			items = append(items, part)
		}
	}
	regexpPath = "^" + strings.Join(items, `\/`) + "$"
	return
}

// ErrorHandler interface to which an errorHandler needs to comply.
//
// Used as a field in the router to override the default RrrorHandler implementation.
// Its responsibility is to generate the http Response when an error occurs. That is,
// when requestContext.Error() gets called.
type ErrorHandler func(res http.ResponseWriter, req *http.Request, err string, code int)

// An implementation of an ErrorHandler so we have one if a custom one
// is not explicitly set.
//
// Note: the request is passed so we can always vary our response depending on the request info.
func defaultErrorHandler(res http.ResponseWriter, req *http.Request, err string, code int) {
	http.Error(res, err, code)
}
