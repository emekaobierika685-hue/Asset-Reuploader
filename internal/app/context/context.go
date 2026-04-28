package context

import (
	"github.com/emekaobierika685-hue/Asset-Reuploader/internal/app/response"
	"github.com/emekaobierika685-hue/Asset-Reuploader/internal/roblox"
)

type Context struct {
	Client          *roblox.Client
	Logger          *logger
	PauseController *pauseController
	Response        *response.Response
}

func New(c *roblox.Client, resp *response.Response) *Context {
	return &Context{
		Client:          c,
		Logger:          newLogger(),
		PauseController: newPauseController(),
		Response:        resp,
	}
}
