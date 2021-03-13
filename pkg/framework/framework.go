package framework

import (
	eventrouter "github.com/genkami/go-slack-event-router"
	interactionrouter "github.com/genkami/go-slack-event-router/interactionrouter"
	"github.com/slack-go/slack"
)

type Command interface {
	Register(*Framework) error
}

type Framework struct {
	EventRouter       *eventrouter.Router
	InteractionRouter *interactionrouter.Router
	Client            *slack.Client
}

func (f *Framework) Use(commands ...Command) error {
	for _, cmd := range commands {
		if err := cmd.Register(f); err != nil {
			return err
		}
	}
	return nil
}
