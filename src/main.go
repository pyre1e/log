package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/fasthttp/router"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/pprofhandler"
)

type LogRequest struct {
	user_id uuid.UUID
}

var conn driver.Conn
var err error

func main() {

	conn, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	r := router.New()

	r.GET("/debug/pprof/{profile:*}", pprofhandler.PprofHandler)
	r.POST("/log", log)

	fmt.Println("starting...")

	if err := fasthttp.ListenAndServe(":80", r.Handler); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func log(ctx *fasthttp.RequestCtx) {
	data := &LogRequest{}

	if err := json.Unmarshal(ctx.Request.Body(), &data); err != nil {
		fmt.Println(err)
	}

	fmt.Fprintln(ctx, data)
}
