package echo

import (
	"context"
	"regexp"

	"github.com/genkami/go-slack-event-router/appmention"
	routererrors "github.com/genkami/go-slack-event-router/errors"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"

	"github.com/genkami/daemon3/pkg/framework"
)

type Handler struct {
	framework *framework.Framework
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(f *framework.Framework) error {
	h.framework = f
	f.EventRouter.OnAppMention(appmention.HandlerFunc(h.HandleEcho), appmention.TextRegexp(regexp.MustCompile(`\becho\b`)))
	return nil
}

var echoPattern = regexp.MustCompile(`\becho[[:space:]]+(.+)`)

func (h *Handler) HandleEcho(ctx context.Context, e *slackevents.AppMentionEvent) error {
	groups := echoPattern.FindStringSubmatch(e.Text)
	if len(groups) < 2 {
		return routererrors.NotInterested
	}
	arg := groups[1]
	if _, _, err := h.framework.Client.PostMessageContext(ctx, e.Channel, slack.MsgOptionText(arg, true)); err != nil {
		return err
	}
	return nil
}
