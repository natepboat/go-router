package api

import (
	"net/http"
	"time"
)

func Home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(time.Now().String()))
}
