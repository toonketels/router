package router

import (
	"net/http"
	"regexp"
)

// RequestHandler
// --------------------------------

// RequestHandler stores info to evaluate if a route can be
// matched, for which params and which HandlerFuncs to dispatch.
type requestHandler struct {
	Path       string
	ParamNames []string
	Regex      *regexp.Regexp
	Tokenized  bool
	Handlers   []http.HandlerFunc
}

// matches checks if the given handler matches the given given string.
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

	// Compare via regexp when the path does contain tokens
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

// middlewareRequestHandler is similar to requestHandler but is mounted.
type middlewareRequestHandler struct {
	MountPath string
	Handle    http.HandlerFunc
	Matcher   *regexp.Regexp
}

// Checks whether the middlewareRequestHandler matches the given path.
func (mReqHandler *middlewareRequestHandler) shouldMount(path string) bool {
	return mReqHandler.Matcher.MatchString(path)
}
