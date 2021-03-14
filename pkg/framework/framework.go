package framework

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sort"

	eventrouter "github.com/genkami/go-slack-event-router"
	"github.com/genkami/go-slack-event-router/appmention"
	interactionrouter "github.com/genkami/go-slack-event-router/interactionrouter"
	"github.com/go-logr/logr"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type Command interface {
	Register(*Framework) error
}

type Framework struct {
	EventRouter       *eventrouter.Router
	InteractionRouter *interactionrouter.Router
	Client            *slack.Client
	Log               logr.Logger
	helps             map[string]string
}

func (f *Framework) Use(commands ...Command) error {
	for _, cmd := range commands {
		if err := cmd.Register(f); err != nil {
			return err
		}
	}
	return nil
}

func (f *Framework) Help(name, description string) {
	if f.helps == nil {
		f.helps = map[string]string{}
		f.EventRouter.OnAppMention(appmention.HandlerFunc(f.HandleHelp),
			appmention.TextRegexp(regexp.MustCompile(`<@.+>[[:space:]]*help[[:space:]]*$`)))
	}
	f.helps[name] = description
}

func (f *Framework) HandleHelp(ctx context.Context, e *slackevents.AppMentionEvent) error {
	textOpt := slack.MsgOptionText(f.buildHelpText(), false)
	if _, _, err := f.Client.PostMessageContext(ctx, e.Channel, textOpt); err != nil {
		f.Log.Error(err, "failed to post message")
		return err
	}
	return nil
}

func (f *Framework) buildHelpText() string {
	buf := bytes.NewBuffer(nil)
	names := make([]string, 0, len(f.helps))
	for name := range f.helps {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(buf, "*%s*: %s\n", name, f.helps[name])
	}
	return buf.String()
}
