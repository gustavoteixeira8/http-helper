package main

import (
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
		return ctx.Status(200).JSON(map[string]string{"id": "1234me", "name": "Gustavo Teixeira"})
	}

	func2 := func(ctx *myserver.Ctx) error {
		if ctx.Params()["id"] != "123456" {
			return ctx.Status(400).JSON("bad request bro")
		}

		return ctx.Next()
	}

	func3 := func(ctx *myserver.Ctx) error {
		if ctx.Params()["token"] != "token-123456" {
			return ctx.Status(400).JSON("bad token bro")
		}

		return ctx.Next()
	}

	server.Handle("/user/me/{id}/{token}", GET, func2, func3, func1)

	// server.Handle("/test", GET, func(ctx *myserver.Ctx) error {
	// 	return ctx.Status(200).JSON(ctx.Path())
	// })

	// server.Handle("/user/", POST, func(w http.ResponseWriter, r *http.Request) (any, error) {
	// 	return "CREATE USER", nil
	// })

	// server.Handle("/user/{id}/{token}", PUT, func(w http.ResponseWriter, r *http.Request) (any, error) {
	// 	return "UPDATE USER", nil
	// })

	log.Fatalln(http.ListenAndServe(":3000", server))
}
