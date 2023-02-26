package api

import (
	"fmt"
	"net/http"

	"github.com/natepboat/go-rest-api/contextKey"
)

func GetUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Context().Value(contextKey.Route{}))
	pathParam := r.Context().Value(contextKey.PathParam{}).(map[string]string)
	w.Write([]byte("getuser: " + pathParam["id"]))
}
