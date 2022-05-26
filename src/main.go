package main

import (
	"fmt"
	"os"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	// "github.com/valyala/fasthttp/pprofhandler"
)

func main() {

	app := CreateApp(&AppOptions{
		Pool: &AppPoolOptions{
			Size: 16,
		},
	})

	rtr := router.New()

	// rtr.GET("/debug/pprof/{profile:*}", pprofhandler.PprofHandler)
	rtr.POST("/log", app.RequestHandler(LogAdd))

	fmt.Println("starting...")

	app.Pool.WaitForConnection()
	InitLog(app, 16)

	fmt.Println("ready")

	if err := fasthttp.ListenAndServe(":80", rtr.Handler); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
