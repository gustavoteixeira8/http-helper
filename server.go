package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var builtinMimeTypesLower = map[string]string{
	".css":  "text/css; charset=utf-8",
	".gif":  "image/gif",
	".htm":  "text/html; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".jpg":  "image/jpeg",
	".js":   "application/javascript",
	".wasm": "application/wasm",
	".pdf":  "application/pdf",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".xml":  "text/xml; charset=utf-8",
}

const (
	_MethodServeStatic = "FILESERVER"
)

func Mime(ext string) string {
	if v, ok := builtinMimeTypesLower[ext]; ok {
		return v
	}
	return mime.TypeByExtension(ext)
}

type StaticOpts struct {
	Path        string
	EmbedFolder embed.FS
}

type DefaultHttpFunc func(c *Ctx) error

type Server struct {
	rootHandler *http.ServeMux
	routes      map[string]map[string][]DefaultHttpFunc
	middlewares []DefaultHttpFunc
}

func (s Server) GetRootHandler() *http.ServeMux {
	return s.rootHandler
}

func (s *Server) Use(middlewares ...DefaultHttpFunc) {
	s.middlewares = append(s.middlewares, middlewares...)
}

func (s *Server) ServeStatic(path string, opts *StaticOpts) error {
	if opts == nil {
		return errors.New("static opts is required")
	}

	fileServerCb := func(ctx *Ctx) error {
		var (
			fileToServe  []byte
			err          error
			absolutePath string
		)

		path := s.formatPathSlash(strings.Split(ctx.Path(), "?")[0])
		path = strings.TrimSuffix(path, "/")
		ext := filepath.Ext(path)

		if ext == "" {
			path = fmt.Sprintf("%s/index.html", path)
			ext = filepath.Ext(path)
		}

		if !reflect.ValueOf(opts.Path).IsZero() {

			if strings.HasPrefix(opts.Path, "./") {
				path = strings.TrimPrefix(path, "/")
				path = fmt.Sprintf("./%s", path)
			}

			absolutePath, err = filepath.Abs(path)

			if err != nil {
				return ctx.Status(http.StatusInternalServerError).JSON(err.Error())
			}

			fileToServe, err = os.ReadFile(absolutePath)

		} else if !reflect.ValueOf(opts.EmbedFolder).IsZero() {
			fileToServe, err = opts.EmbedFolder.ReadFile(path)
		}

		if err != nil {
			return ctx.Status(http.StatusNotFound).JSON(fmt.Sprintf("path not found (%v)\n", err))
		}

		mimeType := Mime(ext)

		ctx.ResponseHeader().Set("Content-Type", mimeType)
		ctx.Write(fileToServe)
		return nil
	}

	s.handle(_MethodServeStatic, path, fileServerCb)

	return nil
}

func (s *Server) handle(method, path string, callback ...DefaultHttpFunc) {
	rootHttpFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
		)

		pathFormatted := s.formatPathSlash(r.URL.Path)

		foundPath := false

		for routeMethod, routePaths := range s.routes {
			if routeMethod == r.Method || routeMethod == _MethodServeStatic {
				_, okWithPathFormatted := routePaths[pathFormatted]
				_, okWithPath := routePaths[path]

				if okWithPathFormatted || okWithPath {
					ctx := NewCtx(w, r)

					for _, c := range routePaths[path] {
						ctx.setNextToFalse()
						err = c(ctx)
						if err != nil {
							ctx.Status(http.StatusInternalServerError).JSON(err.Error())
							return
						}
						if !ctx.getShouldIGoToNext() {
							return
						}
					}

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
	})

	if s.routes[method][path] == nil && len(s.routes[method]) <= 0 {
		s.routes[method] = make(map[string][]DefaultHttpFunc)
	}

	pathFormatted := s.formatPathSlash(path)

	s.routes[method][path] = append(s.routes[method][path], s.middlewares...)
	s.routes[method][pathFormatted] = append(s.routes[method][pathFormatted], s.middlewares...)

	for routeMethod, routePaths := range s.routes {
		_, ok := routePaths[path]
		if ok && routeMethod != method {
			s.routes[method][path] = append(s.routes[method][path], callback...)
			s.routes[method][pathFormatted] = append(s.routes[method][pathFormatted], callback...)
			return
		}
	}

	s.rootHandler.Handle(path, rootHttpFunc)
	s.rootHandler.Handle(pathFormatted, rootHttpFunc)

	s.routes[method][path] = append(s.routes[method][path], callback...)
	s.routes[method][pathFormatted] = append(s.routes[method][pathFormatted], callback...)
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

		dir := filepath.Dir(path)

		for routePath := range s.routes[_MethodServeStatic] {
			if strings.HasPrefix(dir, routePath) {
				correctPath = path
				return
			}
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
		r.Header.Add(fmt.Sprintf("param:%s", key), param)
	}

	if correctPath == "" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(fmt.Sprintf("method not allowed to path %s", r.URL.Path))
		return
	}

	r.URL.Path = correctPath

	s.rootHandler.ServeHTTP(w, r)
}

func (s *Server) Get(path string, callback ...DefaultHttpFunc) {
	s.handle(http.MethodGet, path, callback...)
}

func (s *Server) Options(path string, callback ...DefaultHttpFunc) {
	s.handle(http.MethodOptions, path, callback...)
}

func (s *Server) Post(path string, callback ...DefaultHttpFunc) {
	s.handle(http.MethodGet, path, callback...)
}

func (s *Server) Put(path string, callback ...DefaultHttpFunc) {
	s.handle(http.MethodGet, path, callback...)
}

func (s *Server) Delete(path string, callback ...DefaultHttpFunc) {
	s.handle(http.MethodGet, path, callback...)
}

func (s *Server) Patch(path string, callback ...DefaultHttpFunc) {
	s.handle(http.MethodGet, path, callback...)
}

func NewServer(handler *http.ServeMux) *Server {
	routes := map[string]map[string][]DefaultHttpFunc{}

	return &Server{rootHandler: handler, routes: routes}
}
