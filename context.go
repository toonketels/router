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
	cntxt, _ := requestContextStore.Load(req)
	return cntxt.(*RequestContext)
}

// Next invokes the next HandleFunc in line registered to handle this request.
//
// This is needed when multiple HandleFuncs are registered for a given path
// and allows the creation and use of `middleware`.
func (cntxt *RequestContext) Next(res http.ResponseWriter, req *http.Request) {
	// Don't continue when erring
	if cntxt.inError {
		return
	}
	// For safety reasons, we ensure there is always an empty requestHandler to be
	// called. This to prevent panics when the last requestHandler would call Next.
	// Better safe than sorry.
	var handler http.HandlerFunc
	if len(cntxt.handlers) < cntxt.currentHandler+1 {
		handler = func(res http.ResponseWriter, req *http.Request) {}
	} else {
		handler = cntxt.handlers[cntxt.currentHandler]
	}
	cntxt.currentHandler++
	handler(res, req)
}

// Error allows you to respond with an error message preventing the
// subsequent handlers from being executed.
//
// Note: in case there exist previous requestHandlers and they have code after their
// next call, that code will execute.
// This allows loggers and such to finish what they started (though they can also
// use a defer for that).
func (cntxt *RequestContext) Error(res http.ResponseWriter, req *http.Request, err string, code int) {
	cntxt.inError = true
	cntxt.errorHandler(res, req, err, code)
}

// Set saves a value for the current request.
// The value will not be set if the key already exist.
func (cntxt *RequestContext) Set(key, val interface{}) bool {
	// Lazily create the store
	cntxt.makeStoreIfNotExist()
	if cntxt.store[key] != nil {
		return false
	}
	cntxt.store[key] = val
	return true
}

// ForceSet saves a value for the current request. Unlike Set,
// it will happily override existing data.
func (cntxt *RequestContext) ForceSet(key, val interface{}) {
	// Lazily create the store
	cntxt.makeStoreIfNotExist()
	cntxt.store[key] = val
}

// Get fetches data from the store associated with the current request.
func (cntxt *RequestContext) Get(key interface{}) (val interface{}, ok bool) {
	// Lazily create the store
	cntxt.makeStoreIfNotExist()
	val, ok = cntxt.store[key]
	return
}

// Delete removes a key/value pair from the store.
func (cntxt *RequestContext) Delete(key interface{}) {
	// Lazily create the store
	cntxt.makeStoreIfNotExist()
	delete(cntxt.store, key)
}

// Lazily creates the store if it does not yet exist.
func (cntxt *RequestContext) makeStoreIfNotExist() {
	if cntxt.store == nil {
		cntxt.store = make(map[interface{}]interface{})
	}
}
