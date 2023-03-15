package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type Ctx struct {
	request         *http.Request
	responseWriter  http.ResponseWriter
	shouldIGoToNext bool
	localsVar       map[string]any
}

func (c *Ctx) Locals(key string, value ...any) any {
	localVar, ok := c.localsVar[key]

	if ok && reflect.ValueOf(value).IsZero() {
		return localVar
	}

	c.localsVar[key] = value[0]
	return value
}

func (c *Ctx) Body() ([]byte, error) {
	bodyBytes, err := io.ReadAll(c.request.Body)
	defer c.request.Body.Close()

	return bodyBytes, err
}

func (c *Ctx) Path() string {
	return c.request.URL.Path
}

func (c *Ctx) RequestHeader() *http.Header {
	return &c.request.Header
}

func (c *Ctx) ResponseHeader() *http.Header {
	header := c.responseWriter.Header()
	return &header
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

func (c *Ctx) Files() {
	c.request.ParseMultipartForm(4096)
	fmt.Println(c.request.MultipartForm.File)
	fmt.Println(c.request.MultipartForm.Value)
}

func (c *Ctx) JSON(value any) error {
	c.responseWriter.Header().Set("Content-Type", "application/json")
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

func (c *Ctx) Write(v []byte) error {
	_, err := c.responseWriter.Write(v)
	return err
}

func NewCtx(w http.ResponseWriter, r *http.Request) *Ctx {
	locals := map[string]any{}
	return &Ctx{request: r, responseWriter: w, localsVar: locals}
}
