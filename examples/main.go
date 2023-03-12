package main

import (
	"fmt"
	"log"
	"net/http"

	myserver "github.com/gustavoteixeira8/httphelper"
)

const (
	GET    = http.MethodGet
	POST   = http.MethodPost
	PUT    = http.MethodPut
	DELETE = http.MethodDelete
	PATCH  = http.MethodPatch
)

func main() {
	server := myserver.NewServer(http.NewServeMux())

	func1 := func(ctx *myserver.Ctx) error {
		// fmt.Println(ctx.Params())
		return ctx.Status(200).JSON(fmt.Sprintf("FUNC1 %v", ctx.Locals("KEY")))
	}

	func2 := func(c *myserver.Ctx) error {
		return c.Status(200).JSON(fmt.Sprintf("FUNC2 %v", c.Locals("KEY")))
	}

	middleware := func(ctx *myserver.Ctx) error {
		ctx.Locals("KEY", "VALUE 1")
		return ctx.Next()
	}

	middleware2 := func(ctx *myserver.Ctx) error {
		ctx.Locals("KEY", "VALUE 2")
		return ctx.Next()
	}

	server.Handle("/func1", GET, middleware, func1)
	// server.Use(middleware)
	server.Handle("/func2", GET, middleware2, func2)

	log.Fatalln(http.ListenAndServe(":3000", server))
}
