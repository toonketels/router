package router

import (
	"net/http"
)

// Context
// --------------------------------

// RequestContext contains data related to the current request
type RequestContext struct {
	Params         map[string]string
	inError        bool
	handlers       []http.HandlerFunc
	currentHandler int
	errorHandler   ErrorHandler
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
