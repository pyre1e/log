package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/fasthttp/router"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/pprofhandler"
)

type Event struct {
	EventName string `json:"event_name" validate:"required"`
	EventTxt  string `json:"event_txt" validate:"required"`
}

type LogRequest struct {
	UserId    uuid.UUID `json:"user_id" validate:"required"`
	Timestamp int       `json:"timestamp" validate:"required"`
	Events    []*Event  `json:"events" validate:"required"`
}

var conn driver.Conn
var err error
var validate *validator.Validate

func main() {

	conn, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{"clickhouse:9000"},
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

	validate = validator.New()

	fmt.Println("starting...")

	if err := fasthttp.ListenAndServe(":80", r.Handler); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func log(ctx *fasthttp.RequestCtx) {
	logEntry := LogRequest{}

	if err := json.Unmarshal(ctx.Request.Body(), &logEntry); err != nil {
		ctx.Response.SetStatusCode(400)
		ctx.Response.SetBodyString(err.Error())
		return
	}

	if err := validate.Struct(logEntry); err != nil {
		ctx.Response.SetStatusCode(400)
		ctx.Response.SetBodyString(err.Error())
		return
	}

	for _, event := range logEntry.Events {
		if err := validate.Struct(event); err != nil {
			ctx.Response.SetStatusCode(400)
			ctx.Response.SetBodyString(err.Error())
			return
		}
	}

	go func() {
		cctx := context.Background()
		logId := uuid.New()
		ip, _, _ := net.SplitHostPort(ctx.RemoteAddr().String())

		_, err := conn.Query(cctx,
			"INSERT INTO logs  (id, user_id, timestamp, ip) VALUES ($1, $2, $3, $4);",
			logId, logEntry.UserId, logEntry.Timestamp, ip,
		)
		if err != nil && err != io.EOF {
			fmt.Println("Cannot add log entry", err)
			return
		}

		eventsBatch, err := conn.PrepareBatch(cctx, "INSERT INTO events (id, log_id, type, message)")
		if err != nil {
			fmt.Println("Batch error", err)
			return
		}

		for _, event := range logEntry.Events {
			eventsBatch.Append(uuid.New(), logId, event.EventName, event.EventTxt)
		}

		err = eventsBatch.Send()
		if err != nil {
			fmt.Println("Cannot add events", err)
			return
		}
	}()
}
