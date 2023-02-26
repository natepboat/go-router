# GO ROUTER

Router framework with support of [go-app-env](https://github.com/natepboat/go-app-env) for server properties.

## Properties

| Name  | Data type  | Remark  |
|---|---|---|
| server.port | string | In address format. Ex. ":8080", "127.0.0.1:9000" |
| server.requestTimeout | string | In time.Duration format. Ex. "1m10s" |
| server.writeTimeout | string | In time.Duration format. Ex. "1m10s" |

## URL Parameter

Router read path parameter *name* and *segment location* from configured route that has colon(`:`) prefix in the path segment and put into
request context with key `contextKey.PathParam{}` of type `map[string]string`

Example:
```go
r.AddRoute(httpMethod.GET, "/users/:type/:id", handler)
r.AddRoute(httpMethod.GET, "/users/:id", handler)
r.AddRoute(httpMethod.GET, "/users/:id/items/:itemId", handler)


/**
call GET: /users/employee/123
request context key "contextKey.PathParam{}" will contains
map[type:employee id:123]

call GET: /users/123
request context key "contextKey.PathParam{}" will contains
map[id:123]

call GET: /users/123/items/i-001
request context key "contextKey.PathParam{}" will contains
map[id:123 itemId:i-001]
*/

```

## Trace ID

Router use UUID string for trace id, any match route can access trace id from request context key `contextKey.TraceId{}`.

For client side can access trace id from response header name `x-trace-id`

Example of usage:
```go
package main
//... import
func main() {
	r := router.NewRouter(nil, nil) // default router

    /* customize router
    appEnv := goappenv.NewAppEnv(fsys)
    logger := log.New(os.Stdout, "", log.Ldate)
    r := router.NewRouter(appEnv, logger) 
    */

	r.AddRoute(httpMethod.GET, "/", api.Home)
	r.AddRoute(httpMethod.GET, "/user/:id", api.GetUser)
	server, err := r.NewServer()
	if err != nil {
		log.Fatalln("Cannot create route server", err)
	}
	log.Fatalln(server.ListenAndServe())
}
```
