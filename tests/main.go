package main

import (
	"log"
	"net/http"

	server "github.com/gustavoteixeira8/httphelper"
)

const (
	GET    = http.MethodGet
	POST   = http.MethodPost
	PUT    = http.MethodPut
	DELETE = http.MethodDelete
	PATCH  = http.MethodPatch
)

func main() {
	server := server.NewServer(http.NewServeMux())

	server.Handle("/user/me", GET, func(w http.ResponseWriter, r *http.Request) (any, error) {
		return map[string]string{"id": "1234me", "name": "Gustavo Teixeira"}, nil
	})

	server.Handle("/user/{id}", GET, func(w http.ResponseWriter, r *http.Request) (any, error) {
		return map[string]string{"id": r.Header.Get("id"), "name": "Gustavo Teixeira"}, nil
	})

	server.Handle("/user/", POST, func(w http.ResponseWriter, r *http.Request) (any, error) {
		return "CREATE USER", nil
	})

	server.Handle("/user/{id}/{token}", PUT, func(w http.ResponseWriter, r *http.Request) (any, error) {
		return "UPDATE USER", nil
	})

	log.Fatalln(http.ListenAndServe(":3000", server))
}
