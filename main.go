package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const (
	GET    = http.MethodGet
	POST   = http.MethodPost
	PUT    = http.MethodPut
	DELETE = http.MethodDelete
	PATCH  = http.MethodPatch
)

func main() {
	server := NewServer(http.NewServeMux())

	apiUser := func(w http.ResponseWriter, r *http.Request) (any, error) {
		return "olá mundo", nil
	}

	server.Handle("/user/{id}/update/{token}", PUT, apiUser)

	log.Fatalln(http.ListenAndServe(":3000", server))
}

type Server struct {
	rootHandler *http.ServeMux
	routes      map[string]string
}

type DefaultHttpFunc[OUT any] func(w http.ResponseWriter, r *http.Request) (OUT, error)

func (s *Server) Handle(path, method string, callback DefaultHttpFunc[any]) {
	rootHttpFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = strings.ToUpper(method)

		var (
			out any
			err error
		)

		switch {
		case r.Method == GET && method == GET:
			out, err = callback(w, r)
		case r.Method == POST && method == POST:
			out, err = callback(w, r)
		case r.Method == PUT && method == PUT:
			out, err = callback(w, r)
		case r.Method == DELETE && method == DELETE:
			out, err = callback(w, r)
		case r.Method == PATCH && method == PATCH:
			out, err = callback(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})

	s.rootHandler.Handle(path, rootHttpFunc)

	pathFormatted := s.formatPathSlash(path)

	s.rootHandler.Handle(pathFormatted, rootHttpFunc)

	s.routes[path] = path
	s.routes[pathFormatted] = pathFormatted
}

func (s Server) formatPathSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		return fmt.Sprintf("%s/", path)
	}

	return strings.TrimSuffix(path, "/")
}

func (s Server) clearStringSlice(slice []string) []string {
	newSlice := []string{}

	for _, v := range slice {
		if v != "" && v != " " {
			newSlice = append(newSlice, v)
		}
	}

	return newSlice
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	correctPath := ""

	for _, route := range s.routes {
		pathSplited := s.clearStringSlice(strings.Split(path, "/"))
		routeSplited := s.clearStringSlice(strings.Split(route, "/"))

		if len(pathSplited) != len(routeSplited) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode("path not found")
			return
		}

		notEqualCount := 0
		params := map[string]string{}

		for i := 0; i < len(pathSplited); i++ {
			if pathSplited[i] != routeSplited[i] {

				if strings.Contains(routeSplited[i], "{") && strings.Contains(routeSplited[i], "}") {
					key := strings.Replace(routeSplited[i], "{", "", 1)
					key = strings.Replace(key, "}", "", 1)
					params[key] = string(pathSplited[i])
					continue
				}

				notEqualCount++
			}
		}

		if notEqualCount == 0 {
			correctPath = route
			break
		}
	}

	r.URL.Path = correctPath

	s.rootHandler.ServeHTTP(w, r)
}

func NewServer(handler *http.ServeMux) *Server {
	return &Server{rootHandler: handler, routes: make(map[string]string)}
}
