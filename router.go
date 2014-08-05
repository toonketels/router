package router

import (
	"net/http"
	"regexp"
	"strings"
)

// Stores
// ----------------------

// Store to keep track of the requestContext
var requestContextStore = make(map[*http.Request]*RequestContext)

// Router
// ----------------------

// A Router to register paths and requestHandlers to.
//
// Set a custom NotFoundHandler if you want to override go's default one.
//
// There can be multiple per application, if so, don't forget to pass a
// different pattern to `router.Handle()`.
type Router struct {
	routes          map[string][]*requestHandler
	middleware      []middlewareRequestHandler
	NotFoundHandler http.HandlerFunc                                                       // Specify a custom NotFoundHandler
	ErrorHandler    func(res http.ResponseWriter, req *http.Request, err string, code int) // Specify a custom ErrorHandler
}

// NewRouter creates a router and returns a pointer to it so
// you can start registering routes.
//
// Dont forget to call `router.Handle(pattern)` to actually use
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

// Register a GET path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Get(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("GET", path, handlers...)
}

// Register a POST path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Post(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("POST", path, handlers...)
}

// Register a PUT path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Put(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("PUT", path, handlers...)
}

// Register a DELETE path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Delete(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("DELETE", path, handlers...)
}

// Register a PATCH path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Patch(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("PATCH", path, handlers...)
}

// Register a OPTONS path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Options(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("OPTIONS", path, handlers...)
}

// Register a HEAD path to be handled. Multiple handlers can be passed and
// will be evaluated in order (after the more generic mounted handlerFuncs).
func (router *Router) Head(path string, handlers ...http.HandlerFunc) {
	router.registerRequestHandler("HEAD", path, handlers...)
}

// Mount registers a middleware requestHandler which will be evaluated on each
// path corresponding to the mountPath.
func (router *Router) Mount(mountPath string, handler http.HandlerFunc) {
	mReqHandler := middlewareRequestHandler{
		MountPath: mountPath,
		Handle:    handler,
		Matcher:   regexp.MustCompile(`^\` + mountPath),
	}
	router.middleware = append(router.middleware, mReqHandler)
}

// Handle registers the router for the given pattern in the DefaultServeMux.
// The documentation for ServeMux explains how patterns are matched.
//
// This just delegetes to `http.Handle()` internally.
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
			requestContextStore[req] = cntxt
			// Capture the route params
			cntxt.Params = withParams
			// Attach the handlers to the context
			cntxt.handlers = reqHandler.Handlers
			// Set the errorhandler
			cntxt.errorHandler = router.ErrorHandler
			// Dispatch the first handler,
			// the request is being served.
			cntxt.Next(res, req)
			// Clean up
			delete(requestContextStore, req)
			break
		}
	}

	// Nothing found...
	if unMatched {
		router.notFound(res, req)
	}
}

// Helper function to actually register the requestHandler on the router
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
	middleware := router.middlewareToMount(path)
	// Make the mountedMiddleware the first handlers to be called
	// followed by our registered handlers... keeping everything in order
	handlers = append(middleware, handlers...)

	// Build the regex string to match each incoming request against
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

// Returns all middlewareRequestHandlers that should be mounted for the given path.
func (router *Router) middlewareToMount(path string) (mountedMiddleware []http.HandlerFunc) {
	mountedMiddleware = make([]http.HandlerFunc, 0)
	for _, mReqHandler := range router.middleware {
		if mReqHandler.shouldMount(path) {
			mountedMiddleware = append(mountedMiddleware, mReqHandler.Handle)
		}
	}
	return
}

// Context
// --------------------------------

// RequestContext contains data related to the current request
type RequestContext struct {
	Params         map[string]string
	inError        bool
	handlers       []http.HandlerFunc
	currentHandler int
	errorHandler   func(res http.ResponseWriter, req *http.Request, err string, code int)
	store          map[interface{}]interface{}
}

// Context returns a pointer to the RequestContext for the current request.
func Context(req *http.Request) *RequestContext {
	return requestContextStore[req]
}

// RequestContext.Next() allows a http.HandleFunc to invoke the next HandleFunc.
// This is useful when multiple HandleFuncs are registered for a given path
// and allows the creation and use of `middleware`.
func (cntxt *RequestContext) Next(res http.ResponseWriter, req *http.Request) {
	// Dont continue when erroring
	if cntxt.inError {
		return
	}
	// For safety reasons, we ensur there is always an emtpy requestHandler to be
	// called. This to prevent panics when the last requestHandler would call next.
	// Wont happen often but better safe than sorry.
	var handler http.HandlerFunc
	if len(cntxt.handlers) < cntxt.currentHandler+1 {
		handler = func(res http.ResponseWriter, req *http.Request) {}
	} else {
		handler = cntxt.handlers[cntxt.currentHandler]
	}
	cntxt.currentHandler++
	handler(res, req)
}

// requestContext.Error() allows you to respond with an error message preventing the
// subsequent handlers from being executed.
//
// Note: in case there exist previous requestHandlers and they have code after their
// next call, that code will get execute.
// This allows loggers and such to finish what they started (though they can also
// use a defer for that).
func (cntxt *RequestContext) Error(res http.ResponseWriter, req *http.Request, err string, code int) {
	cntxt.inError = true
	cntxt.errorHandler(res, req, err, code)
}

// requestContext.Set() allows you to save a value for the current request.
// Won't set the value if the key is already used.
func (cntxt *RequestContext) Set(key, val interface{}) bool {
	// Lazely create the store
	cntxt.makeStoreIfNotExist()
	if cntxt.store[key] != nil {
		return false
	}
	cntxt.store[key] = val
	return true
}

// requestContext.ForceSet() allows you to save a value for the current request.
// Unlike Set(), it will happely override exisitng data.
func (cntxt *RequestContext) ForceSet(key, val interface{}) {
	// Lazely create the store
	cntxt.makeStoreIfNotExist()
	cntxt.store[key] = val
}

// requestContext.Get() allows you to fetch data from the store.
func (cntxt *RequestContext) Get(key interface{}) (val interface{}, ok bool) {
	// Lazely create the store
	cntxt.makeStoreIfNotExist()
	val, ok = cntxt.store[key]
	return
}

// requestContext.Delete() allows to delete key value pairs from the store.
func (cntxt *RequestContext) Delete(key interface{}) {
	// Lazely create the store
	cntxt.makeStoreIfNotExist()
	delete(cntxt.store, key)
}

// Lazely creates the store if it does not yet exist
func (cntxt *RequestContext) makeStoreIfNotExist() {
	if cntxt.store == nil {
		cntxt.store = make(map[interface{}]interface{})
	}
}

// RequestHandler
// --------------------------------

// RequestHandler stores info to evaluate if a route can be
// matched, for which params and which handlerFunc to dispatch.
type requestHandler struct {
	Path       string
	ParamNames []string
	Regex      *regexp.Regexp
	Tokenized  bool
	Handlers   []http.HandlerFunc
}

// requestHandler.matches checks if the given handler matches the given given string.
//
// It will also return to which uservalues the params evaluate for this path.
func (reqHandler *requestHandler) matches(path string) (isAMatch bool, withParams map[string]string) {
	withParams = make(map[string]string)
	isAMatch = false

	// Compare strings only when we know the path registered
	// does not contain tokens
	if !reqHandler.Tokenized {
		isAMatch = reqHandler.Path == path
		return
	}

	// Compare via regex when the path does contain tokens
	matches := reqHandler.Regex.FindAllStringSubmatch(path, -1)
	// Only try to find the params if we have a match
	if isAMatch = len(matches) != 0; isAMatch {
		for i, paramName := range reqHandler.ParamNames {
			withParams[paramName] = matches[0][i+1]
		}
	}
	return
}

// MiddlewareRequestHandler
// --------------------------------

type middlewareRequestHandler struct {
	MountPath string
	Handle    http.HandlerFunc
	Matcher   *regexp.Regexp
}

// Checks whether the middlewareRequestHandler matches the given path.
func (mReqHandler *middlewareRequestHandler) shouldMount(path string) bool {
	return mReqHandler.Matcher.MatchString(path)
}

// Private helper funcs
// ---------------------------

// Some paths use tokens like "/user/:userid" where "userid" is the token.
//
// This function builds a string to be compiled as a regexp to match those
// paths and returns the names of the parameters found in the route.
func buildRegexpFor(path string) (regexpPath string, withParamNames []string) {
	parts := strings.Split(path, "/")
	items := make([]string, 0)
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

// An implementation of an errorHandler so we have one if a custom one
// is not explicitly set.
//
// Note: the request is passed so we can always very our response depending on the request info.
func defaultErrorHandler(res http.ResponseWriter, req *http.Request, err string, code int) {
	http.Error(res, err, code)
}
