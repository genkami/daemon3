package holodule

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/genkami/go-slack-event-router/appmention"
	colly "github.com/gocolly/colly/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"

	"github.com/genkami/daemon3/pkg/framework"
)

const UserAgent = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.82 Safari/537.36`

type Handler struct {
	framework           *framework.Framework
	mu                  sync.Mutex
	lastSchedule        Schedule
	lastScheduleUpdated *time.Time
}

type Schedule struct {
	Items []Item
}

type Item struct {
	Time string
	Name string
	Icon string
	URL  string
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(f *framework.Framework) error {
	h.framework = f
	f.EventRouter.OnAppMention(appmention.HandlerFunc(h.HandleHolodule), appmention.TextRegexp(regexp.MustCompile(`\bholodule\b`)))
	f.Help("holodule", "show hololive schedule")
	return nil
}

func (h *Handler) HandleHolodule(ctx context.Context, e *slackevents.AppMentionEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var err error
	if h.lastScheduleUpdated == nil || tooOld(*h.lastScheduleUpdated) {
		h.lastSchedule.Items, err = h.fetchSchedule()
		if err != nil {
			h.framework.Log.Error(err, "failed to update schedule")
			return err
		}
		now := time.Now()
		h.lastScheduleUpdated = &now
	}
	err = h.postSchedule(ctx, e.Channel)
	if err != nil {
		h.framework.Log.Error(err, "failed to post schedule")
		return err
	}
	return nil
}

func (h *Handler) postSchedule(ctx context.Context, channel string) error {
	blocks := make([]slack.Block, 0, 20) // random large number
	blocks = append(blocks,
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				"*Schedule*",
				false, false,
			)))
	for i := range h.lastSchedule.Items {
		item := &h.lastSchedule.Items[i]
		blocks = append(blocks,
			slack.NewSectionBlock(
				slack.NewTextBlockObject(
					slack.MarkdownType,
					fmt.Sprintf("*%s* <%s|%s> %s", item.Time, item.URL, item.Name, item.Icon),
					false, false,
				),
				nil, nil,
			),
		)
	}
	_, _, err := h.framework.Client.PostMessageContext(ctx, channel, slack.MsgOptionBlocks(blocks...))
	return err
}

func (h *Handler) fetchSchedule() ([]Item, error) {
	items := make([]Item, 0, 20) // random large number

	c := colly.NewCollector(colly.UserAgent(UserAgent))
	c.OnHTML(`a[href^="https://www.youtube.com/watch"]`, func(e *colly.HTMLElement) {
		items = append(items, parseItem(e))
	})
	err := c.Visit("https://schedule.hololive.tv/simple")
	if err != nil {
		return nil, err
	}

	return items, nil
}

var spaces = regexp.MustCompile(`[[:space:]]+`)

func parseItem(e *colly.HTMLElement) Item {
	words := spaces.Split(e.Text, 10) // random large number
	item := Item{URL: e.Attr("href")}
	i := 0
	for _, w := range words {
		if w == "" {
			continue
		}
		switch i {
		case 0:
			item.Time = w
		case 1:
			item.Name = w
		case 2:
			item.Icon = w
		default:
			break
		}
		i++
	}
	return item
}

func tooOld(t time.Time) bool {
	return t.Before(time.Now().Add(-1 * time.Hour))
}
