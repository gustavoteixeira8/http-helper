package server

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

/* ------------------------------------------------------- */

type Ctx struct {
	request         *http.Request
	responseWriter  http.ResponseWriter
	shouldIGoToNext bool
}

func (c *Ctx) Body() ([]byte, error) {
	bodyBytes, err := io.ReadAll(c.request.Body)
	defer c.request.Body.Close()

	return bodyBytes, err
}

func (c *Ctx) Path() string {
	return c.request.URL.Path
}

func (c *Ctx) Header() http.Header {
	return c.request.Header
}

func (c *Ctx) Host() string {
	return c.request.Host
}

func (c *Ctx) Method() string {
	return c.request.Method
}

func (c *Ctx) Proto() string {
	return c.request.Proto
}

func (c *Ctx) MultipartForm() multipart.Form {
	return *c.request.MultipartForm
}

func (c Ctx) JSON(value any) error {
	return json.NewEncoder(c.responseWriter).Encode(value)
}

func (c *Ctx) AddCookie(cookies ...*http.Cookie) *Ctx {
	for _, cookie := range cookies {
		c.request.AddCookie(cookie)
	}

	return c
}

func (c *Ctx) Status(status int) *Ctx {
	c.responseWriter.WriteHeader(status)
	return c
}

func (c *Ctx) GetRootRequest() *http.Request {
	return c.request
}

func (c *Ctx) Cookies() []*http.Cookie {
	return c.request.Cookies()
}

func (c *Ctx) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

func (c *Ctx) UserAgent() string {
	return c.request.UserAgent()
}

func (c *Ctx) Redirect(url string, statusCode ...int) {
	if len(statusCode) == 0 {
		statusCode = append(statusCode, http.StatusMovedPermanently)
	}
	http.Redirect(c.responseWriter, c.request, url, statusCode[0])
}

func (c *Ctx) Query() url.Values {
	return c.request.URL.Query()
}

func (c *Ctx) Params() map[string]string {
	params := map[string]string{}

	for key, value := range c.request.Header {
		if strings.Contains(key, "param:") && len(value) > 0 {
			key = strings.TrimPrefix(key, "param:")
			params[key] = value[0]
		}
	}

	return params
}

func (c *Ctx) Next() error {
	c.shouldIGoToNext = true
	return nil
}

func (c *Ctx) setNextToFalse() {
	c.shouldIGoToNext = false
}

func (c *Ctx) getShouldIGoToNext() bool {
	return c.shouldIGoToNext
}

func NewCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	return &Ctx{request: r, responseWriter: w}
}

/* ------------------------------------------------------- */

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

func (s *Server) Handle(path, method string, callback ...DefaultHttpFunc) {
	rootHttpFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		var (
			err error
		)

		pathFormatted := s.formatPathSlash(r.URL.Path)

		foundPath := false

		for routeMethod, routePaths := range s.routes {
			if routeMethod == r.Method {
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

func NewServer(handler *http.ServeMux) *Server {
	routes := map[string]map[string][]DefaultHttpFunc{}

	return &Server{rootHandler: handler, routes: routes}
}
