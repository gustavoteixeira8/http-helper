package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type DefaultHttpFunc[OUT any] func(w http.ResponseWriter, r *http.Request) (OUT, error)

type Server struct {
	rootHandler *http.ServeMux
	routes      map[string]map[string]DefaultHttpFunc[any]
}

func (s Server) GetRootHandler() *http.ServeMux {
	return s.rootHandler
}

func (s *Server) Handle(path, method string, callback DefaultHttpFunc[any]) {
	rootHttpFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		var (
			out any
			err error
		)

		pathFormatted := s.formatPathSlash(r.URL.Path)

		foundPath := false

		for routeMethod, routePaths := range s.routes {
			if routeMethod == r.Method {
				_, okWithPathFormatted := routePaths[pathFormatted]
				_, okWithPath := routePaths[path]

				if okWithPathFormatted || okWithPath {
					out, err = routePaths[path](w, r)
					foundPath = true
					break
				}
			}
		}

		if !foundPath {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode("method not allowed")
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err.Error())
			return
		}

		json.NewEncoder(w).Encode(out)
	})

	if s.routes[method][path] == nil && len(s.routes[method]) <= 0 {
		s.routes[method] = make(map[string]DefaultHttpFunc[any])
	}

	pathFormatted := s.formatPathSlash(path)

	for routeMethod, routePaths := range s.routes {
		_, ok := routePaths[path]
		if ok {
			if routeMethod != method {
				s.routes[method][path] = callback
				s.routes[method][pathFormatted] = callback
			}
			return
		}
	}

	s.rootHandler.Handle(path, rootHttpFunc)
	s.rootHandler.Handle(pathFormatted, rootHttpFunc)

	s.routes[method][path] = callback
	s.routes[method][pathFormatted] = callback
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

func (s Server) pathHasParameter(path string) bool {
	return strings.Contains(path, "{") && strings.Contains(path, "}")
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	var (
		correctPath = ""
		routeParams map[string]string
	)

	func() {
		_, ok := s.routes[r.Method][path]

		if ok {
			correctPath = path
			return
		}

		for routePath := range s.routes[r.Method] {
			pathSplited := s.clearStringSlice(strings.Split(path, "/"))
			routeSplited := s.clearStringSlice(strings.Split(routePath, "/"))

			notEqualCount := 0
			params := map[string]string{}

			routeSplitedLen := len(routeSplited) - 1

			for i := 0; i < len(pathSplited); i++ {
				if routeSplitedLen < i {
					break
				}

				if pathSplited[i] != routeSplited[i] {
					if s.pathHasParameter(routeSplited[i]) {
						key := strings.Replace(routeSplited[i], "{", "", 1)
						key = strings.Replace(key, "}", "", 1)
						params[key] = string(pathSplited[i])
						continue
					}

					notEqualCount++
				}
			}

			if notEqualCount == 0 && len(pathSplited) == len(routeSplited) {
				correctPath = routePath
				routeParams = params
				return
			}
		}
	}()

	for key, param := range routeParams {
		r.Header.Set(key, param)
	}

	if correctPath == "" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(fmt.Sprintf("method not allowed to path %s", r.URL.Path))
		return
	}

	r.URL.Path = correctPath

	s.rootHandler.ServeHTTP(w, r)
}

func NewServer(handler *http.ServeMux) *Server {
	routes := map[string]map[string]DefaultHttpFunc[any]{}

	return &Server{rootHandler: handler, routes: routes}
}
