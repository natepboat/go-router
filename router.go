package gorouter

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	goappenv "github.com/natepboat/go-app-env"
	"github.com/natepboat/go-rest-api/contextKey"
	"github.com/natepboat/go-rest-api/httpMethod"
)

type routeConfig struct {
	httpMethod   httpMethod.HttpMethod
	path         string
	pathSegments []string
	handler      http.HandlerFunc
}

type router struct {
	routes []*routeConfig
	appenv *goappenv.AppEnv
	logger *log.Logger
}

func NewRouter(appenv *goappenv.AppEnv, logger *log.Logger) *router {
	var routerLogger *log.Logger
	if logger == nil {
		routerLogger = log.Default()
	} else {
		routerLogger = logger
	}

	return &router{
		routes: make([]*routeConfig, 0),
		appenv: appenv,
		logger: routerLogger,
	}
}

func (r *router) AddRoute(method httpMethod.HttpMethod, path string, handler http.HandlerFunc) {
	r.routes = append(r.routes, &routeConfig{
		httpMethod:   method,
		path:         path,
		pathSegments: strings.Split(strings.TrimRight(path, "/"), "/"),
		handler:      handler,
	})
}

func (r *router) handle(respWriter http.ResponseWriter, req *http.Request) {
	var targetRoute *routeConfig
	var paramMapCtx map[string]string
	requestSegment := strings.Split(strings.TrimRight(req.URL.Path, "/"), "/")
	requestSegmentLen := len(requestSegment)

	for _, route := range r.routes {
		routeSegments := route.pathSegments
		if len(routeSegments) != requestSegmentLen {
			continue
		}

		isPathMatch, paramMap := isMatchPath(requestSegment, routeSegments)

		if isPathMatch && isMethodMatch(req, route) {
			targetRoute = route
			paramMapCtx = paramMap
			break
		}
	}

	if targetRoute != nil {
		traceId := uuid.NewString()
		ctx := context.WithValue(req.Context(), contextKey.PathParam{}, paramMapCtx)
		ctx = context.WithValue(ctx, contextKey.Route{}, *targetRoute)
		ctx = context.WithValue(ctx, contextKey.TraceId{}, traceId)

		r.logger.Printf("[trace-id=%s] Route: %s %s\n", traceId, req.Method, req.URL)
		respWriter.Header().Add("x-trace-id", traceId)
		targetRoute.handler(respWriter, req.WithContext(ctx))
	} else {
		r.logger.Printf("Route not found: %s %s\n", req.Method, req.URL)
		respWriter.WriteHeader(http.StatusNotFound)
	}
}

func (r *router) NewServer() (*http.Server, error) {
	readTimeout, err := time.ParseDuration(goappenv.ConfigOrDefault(r.appenv, "server.readTimeout", "1m").(string))
	if err != nil {
		return nil, errors.New("server.readTimeout invalid !!, required string format of time.Duration")
	}

	writeTimeout, err := time.ParseDuration(goappenv.ConfigOrDefault(r.appenv, "server.writeTimeout", "1m").(string))
	if err != nil {
		return nil, errors.New("server.writeTimeout invalid !!, required string format of time.Duration")
	}

	return &http.Server{
		Addr:         goappenv.ConfigOrDefault(r.appenv, "server.port", ":8080").(string),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		ErrorLog:     r.logger,
		Handler:      http.HandlerFunc(r.handle),
	}, nil
}

func isMatchPath(requestSegments []string, routeSegments []string) (bool, map[string]string) {
	paramMap := map[string]string{}
	totalMatch := 0

	for i := 0; i < len(requestSegments); i++ {
		routeSegment := routeSegments[i]
		requestSegment := requestSegments[i]
		isParam := strings.HasPrefix(routeSegment, ":")

		if isParam && len(strings.Trim(requestSegment, " ")) > 0 {
			paramMap[strings.TrimPrefix(routeSegment, ":")] = requestSegment
			totalMatch++
		} else if strings.EqualFold(routeSegment, requestSegments[i]) {
			totalMatch++
		} else {
			break
		}
	}

	return totalMatch == len(routeSegments), paramMap
}

func isMethodMatch(req *http.Request, route *routeConfig) bool {
	return strings.EqualFold(req.Method, string(route.httpMethod))
}
