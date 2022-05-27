package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type event struct {
	EventName string `json:"event_name" validate:"required"`
	EventTxt  string `json:"event_txt" validate:"required"`
}

type logRequest struct {
	UserId    uuid.UUID `json:"user_id" validate:"required"`
	Timestamp int       `json:"timestamp" validate:"required"`
	Events    []*event  `json:"events" validate:"required"`
}

type channelMessage struct {
	logEntry logRequest
	ip       string
}

var channel = make(chan *channelMessage)

func InitLog(app *App, workers int) {
	for i := 0; i < workers; i++ {
		go LogAddWorker(app)
	}
}

func LogAdd(ctx *fasthttp.RequestCtx, app *App) {
	logEntry := logRequest{}

	if err := json.Unmarshal(ctx.Request.Body(), &logEntry); err != nil {
		ctx.Response.SetStatusCode(400)
		ctx.Response.SetBodyString(err.Error())
		return
	}

	if err := app.Validate.Struct(logEntry); err != nil {
		ctx.Response.SetStatusCode(400)
		ctx.Response.SetBodyString(err.Error())
		return
	}

	for _, event := range logEntry.Events {
		if err := app.Validate.Struct(event); err != nil {
			ctx.Response.SetStatusCode(400)
			ctx.Response.SetBodyString(err.Error())
			return
		}
	}

	go func() {
		ip, _, _ := net.SplitHostPort(ctx.RemoteAddr().String())

		channel <- &channelMessage{
			logEntry: logEntry,
			ip:       ip,
		}
	}()
}

func LogAddWorker(app *App) {

	qctx := context.Background()

	pc, _, err := app.Pool.Acquire()
	if err != nil {
		fmt.Println("Connection pool error", err)
		return
	}

	for {
		msg := <-channel
		logId := uuid.New()

		_, err = pc.conn.Query(qctx,
			"INSERT INTO logs  (id, user_id, timestamp, ip) VALUES ($1, $2, $3, $4);",
			logId, msg.logEntry.UserId, msg.logEntry.Timestamp, msg.ip,
		)
		if err != nil && err != io.EOF {
			fmt.Println("Cannot add log entry", err)
			return
		}

		if len(msg.logEntry.Events) == 0 {
			continue
		}

		eventsBatch, err := pc.conn.PrepareBatch(qctx, "INSERT INTO events (id, log_id, type, message)")
		if err != nil {
			fmt.Println("Batch error", err)
			return
		}

		for _, event := range msg.logEntry.Events {
			eventsBatch.Append(uuid.New(), logId, event.EventName, event.EventTxt)
		}

		err = eventsBatch.Send()
		if err != nil {
			fmt.Println("Cannot add events", err)
			return
		}
	}
}
