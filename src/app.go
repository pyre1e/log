package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/valyala/fasthttp"
)

type App struct {
	Pool     *AppPool
	Validate *validator.Validate
}

type AppPoolOptions struct {
	Size     int
	Lifetime int
}

type AppOptions struct {
	Pool *AppPoolOptions
}

type RequestHandler func(ctx *fasthttp.RequestCtx, app *App)

func CreateApp(options *AppOptions) *App {
	app := &App{
		Pool:     &AppPool{},
		Validate: validator.New(),
	}
	app.Pool.Init(options.Pool.Size, options.Pool.Lifetime)

	return app
}

func (app *App) RequestHandler(handler RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		handler(ctx, app)
	}
}
