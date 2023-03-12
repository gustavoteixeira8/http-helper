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
		return ctx.Status(200).JSON("FUNC1")
	}

	func2 := func(ctx *myserver.Ctx) error {
		fmt.Println("ROUTE MID")
		return ctx.Next()
	}

	middleware := func(ctx *myserver.Ctx) error {
		fmt.Println("MIDDLEWARE QUE PASSA EM TODAS AS REQUESTS")
		if ctx.Params()["id"] != "123456" {
			return ctx.Status(400).JSON("bad request bro from mid")
		}

		// ctx.Status(500).JSON("não vou passar para o próximo handler")
		return ctx.Next()
	}

	server.Handle("/func1", GET, func1)
	server.Use(middleware)
	server.Handle("/func2/{id}", GET, func2, func(c *myserver.Ctx) error {
		return c.Status(200).JSON("ROUTE END")
	})

	server.Handle("/func2/{id}", POST, func2, func(c *myserver.Ctx) error {
		return c.Status(200).JSON("ROUTE POST END")
	})

	log.Fatalln(http.ListenAndServe(":3000", server))
}
