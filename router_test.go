package gorouter

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"testing/fstest"

	goappenv "github.com/natepboat/go-app-env"
	"github.com/natepboat/go-router/contextKey"
	"github.com/natepboat/go-router/httpMethod"
	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	t.Run("init routes", func(t *testing.T) {
		r := NewRouter(nil, nil)

		assert.NotNil(t, r.routes)
		assert.Empty(t, r.routes)
	})

	t.Run("use default log if logger not provided", func(t *testing.T) {
		r := NewRouter(nil, nil)

		assert.NotNil(t, r.logger)
	})

	t.Run("use provided logger", func(t *testing.T) {
		logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
		r := NewRouter(nil, logger)

		assert.Same(t, logger, r.logger)
	})
}

func TestAddRoute(t *testing.T) {
	noopHandler := func(w http.ResponseWriter, r *http.Request) {}
	r := NewRouter(nil, nil)
	r.AddRoute(httpMethod.GET, "/data", noopHandler)
	r.AddRoute(httpMethod.GET, "/data/:id", noopHandler)
	r.AddRoute(httpMethod.GET, "/data/:id/item", noopHandler)
	r.AddRoute(httpMethod.POST, "/data", noopHandler)
	r.AddRoute(httpMethod.PUT, "/data/:id", noopHandler)
	r.AddRoute(httpMethod.DELETE, "/data/:id", noopHandler)

	assert.Equal(t, 6, len(r.routes))
}

func TestHandle(t *testing.T) {
	t.Run("not match", testHandleNotMatch)
	t.Run("match", testHandleMatch)
}

func TestNewServer(t *testing.T) {
	t.Run("default server", func(t *testing.T) {
		r := NewRouter(nil, nil)
		server, _ := r.NewServer()

		assert.Equal(t, ":8080", server.Addr)
		assert.Equal(t, "1m0s", server.ReadTimeout.String())
		assert.Equal(t, "1m0s", server.WriteTimeout.String())
		assert.NotNil(t, server.ErrorLog)
		assert.NotNil(t, server.Handler.ServeHTTP)
	})

	t.Run("customize server with invalid readTimeout", func(t *testing.T) {
		fsys := fstest.MapFS{
			path.Join("resources", "config.json"): &fstest.MapFile{
				Data: []byte(`{"server":{"port":":9000","readTimeout":"fiveMinute","writeTimeout":"10m"}}`),
			},
		}
		logger := log.New(os.Stdout, "", log.Ldate)

		appenv := goappenv.NewAppEnv(fsys)
		r := NewRouter(appenv, logger)
		server, err := r.NewServer()

		assert.Nil(t, server)
		assert.NotNil(t, err)
	})

	t.Run("customize server with invalid writeTimeout", func(t *testing.T) {
		fsys := fstest.MapFS{
			path.Join("resources", "config.json"): &fstest.MapFile{
				Data: []byte(`{"server":{"port":":9000","readTimeout":"5m","writeTimeout":"tenMinutes"}}`),
			},
		}
		logger := log.New(os.Stdout, "", log.Ldate)

		appenv := goappenv.NewAppEnv(fsys)
		r := NewRouter(appenv, logger)
		server, err := r.NewServer()

		assert.Nil(t, server)
		assert.NotNil(t, err)
	})

	t.Run("customize server", func(t *testing.T) {
		fsys := fstest.MapFS{
			path.Join("resources", "config.json"): &fstest.MapFile{
				Data: []byte(`{"server":{"port":":9000","readTimeout":"5m","writeTimeout":"10m"}}`),
			},
		}
		logger := log.New(os.Stdout, "", log.Ldate)

		appenv := goappenv.NewAppEnv(fsys)
		r := NewRouter(appenv, logger)
		server, _ := r.NewServer()

		assert.Equal(t, ":9000", server.Addr)
		assert.Equal(t, "5m0s", server.ReadTimeout.String())
		assert.Equal(t, "10m0s", server.WriteTimeout.String())
		assert.Same(t, logger, server.ErrorLog)
		assert.NotNil(t, server.Handler.ServeHTTP)
	})
}

func testHandleNotMatch(t *testing.T) {
	noopHandler := func(w http.ResponseWriter, r *http.Request) {}
	r := NewRouter(nil, nil)
	r.AddRoute(httpMethod.GET, "/data", noopHandler)
	r.AddRoute(httpMethod.POST, "/data", noopHandler)

	t.Run("not match path", func(t *testing.T) {
		respWriter := handleHttp(t, r, "PUT", "/item", strings.NewReader("put body"))

		assert.Equal(t, 404, respWriter.Code)
		assert.Equal(t, 0, len(respWriter.Body.Bytes()))
	})

	t.Run("match path but not method", func(t *testing.T) {
		respWriter := handleHttp(t, r, "PUT", "/data", strings.NewReader("put body"))

		assert.Equal(t, 404, respWriter.Code)
		assert.Equal(t, 0, len(respWriter.Body.Bytes()))
	})
}

func testHandleMatch(t *testing.T) {
	successHandler := func(w http.ResponseWriter, r *http.Request) {
		routePath := r.Context().Value(contextKey.Route{}).(routeConfig).path
		pathParam := r.Context().Value(contextKey.PathParam{}).(map[string]string)

		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("routePath:%s|pathParam:%s", routePath, pathParam)))
	}
	noContentHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	r := NewRouter(nil, nil)
	r.AddRoute(httpMethod.GET, "/", successHandler)
	r.AddRoute(httpMethod.GET, "/data", successHandler)
	r.AddRoute(httpMethod.POST, "/data", successHandler)
	r.AddRoute(httpMethod.GET, "/data/:id", successHandler)
	r.AddRoute(httpMethod.PUT, "/data/:id", noContentHandler)
	r.AddRoute(httpMethod.DELETE, "/data/:id", noContentHandler)
	r.AddRoute(httpMethod.GET, "/data/:id/item", successHandler)
	r.AddRoute(httpMethod.GET, "/data/:id/:typeId", successHandler)

	t.Run("response with x-tract-id header", func(t *testing.T) {
		respWriter := handleHttp(t, r, "GET", "", nil)

		assert.Equal(t, 200, respWriter.Code)
		assert.NotEmpty(t, respWriter.Header().Get("x-trace-id"))
	})

	t.Run("root path without trailing slash", func(t *testing.T) {
		respWriter := handleHttp(t, r, "GET", "", nil)

		assert.Equal(t, 200, respWriter.Code)
		assert.Equal(t, "routePath:/|pathParam:map[]", respWriter.Body.String())
	})

	t.Run("root path with trailing slash", func(t *testing.T) {
		respWriter := handleHttp(t, r, "GET", "/", nil)

		assert.Equal(t, 200, respWriter.Code)
		assert.Equal(t, "routePath:/|pathParam:map[]", respWriter.Body.String())
	})

	t.Run("nested path without trailing slash", func(t *testing.T) {
		testcases := []struct {
			respWriter   *httptest.ResponseRecorder
			expectStatus int
			expectBody   string
		}{
			{
				respWriter:   handleHttp(t, r, "GET", "/data", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data|pathParam:map[]",
			},
			{
				respWriter:   handleHttp(t, r, "POST", "/data", strings.NewReader("req body")),
				expectStatus: 200,
				expectBody:   "routePath:/data|pathParam:map[]",
			},
			{
				respWriter:   handleHttp(t, r, "GET", "/data/DAT-123_abc", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data/:id|pathParam:map[id:DAT-123_abc]",
			},
			{
				respWriter:   handleHttp(t, r, "PUT", "/data/DAT-0099", strings.NewReader("req body")),
				expectStatus: 204,
				expectBody:   "",
			},
			{
				respWriter:   handleHttp(t, r, "DELETE", "/data/DAT-001", nil),
				expectStatus: 204,
				expectBody:   "",
			},
			{
				respWriter:   handleHttp(t, r, "GET", "/data/DAT-123_abc/item", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data/:id/item|pathParam:map[id:DAT-123_abc]",
			},
			{
				respWriter:   handleHttp(t, r, "GET", "/data/DAT-123_abc/type_i0123", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data/:id/:typeId|pathParam:map[id:DAT-123_abc typeId:type_i0123]",
			},
		}

		for _, tc := range testcases {
			assert.Equal(t, tc.expectStatus, tc.respWriter.Code)
			assert.Equal(t, tc.expectBody, tc.respWriter.Body.String())
		}
	})

	t.Run("nested path with trailing slash", func(t *testing.T) {
		testcases := []struct {
			respWriter   *httptest.ResponseRecorder
			expectStatus int
			expectBody   string
		}{
			{
				respWriter:   handleHttp(t, r, "GET", "/data/", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data|pathParam:map[]",
			},
			{
				respWriter:   handleHttp(t, r, "POST", "/data/", strings.NewReader("req body")),
				expectStatus: 200,
				expectBody:   "routePath:/data|pathParam:map[]",
			},
			{
				respWriter:   handleHttp(t, r, "GET", "/data/DAT-123_abc/", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data/:id|pathParam:map[id:DAT-123_abc]",
			},
			{
				respWriter:   handleHttp(t, r, "PUT", "/data/DAT-0099/", strings.NewReader("req body")),
				expectStatus: 204,
				expectBody:   "",
			},
			{
				respWriter:   handleHttp(t, r, "DELETE", "/data/DAT-001/", nil),
				expectStatus: 204,
				expectBody:   "",
			},
			{
				respWriter:   handleHttp(t, r, "GET", "/data/DAT-123_abc/item/", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data/:id/item|pathParam:map[id:DAT-123_abc]",
			},
			{
				respWriter:   handleHttp(t, r, "GET", "/data/DAT-123_abc/i0123/", nil),
				expectStatus: 200,
				expectBody:   "routePath:/data/:id/:typeId|pathParam:map[id:DAT-123_abc typeId:i0123]",
			},
		}

		for _, tc := range testcases {
			assert.Equal(t, tc.expectStatus, tc.respWriter.Code)
			assert.Equal(t, tc.expectBody, tc.respWriter.Body.String())
		}
	})
}

func handleHttp(t *testing.T, r *router, method string, path string, body io.Reader) *httptest.ResponseRecorder {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	respWriter := httptest.NewRecorder()
	handler := http.HandlerFunc(r.handle)
	handler.ServeHTTP(respWriter, req)

	return respWriter
}
