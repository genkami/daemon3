package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/genkami/daemon3/pkg/framework"
	"github.com/genkami/daemon3/pkg/handlers/echo"
	eventrouter "github.com/genkami/go-slack-event-router"
	"github.com/genkami/go-slack-event-router/interactionrouter"
	"github.com/slack-go/slack"
)

type Params struct {
	BindAddr      string
	SigningSecret string
	BotToken      string
}

func (p *Params) ParseFlags() {
	flag.StringVar(&p.BindAddr, "addr", "", "bind address")
	flag.Parse()

	p.SigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
	if p.SigningSecret == "" {
		fmt.Fprintln(os.Stderr, "SLACK_SIGNING_SECRET must be set")
		os.Exit(1)
	}
	p.BotToken = os.Getenv("SLACK_BOT_TOKEN")
	if p.BotToken == "" {
		fmt.Fprintln(os.Stderr, "SLACK_BOT_TOKEN must be set")
		os.Exit(1)
	}
}

func main() {
	var p Params
	p.ParseFlags()
	eventRouter, err := eventrouter.New(eventrouter.WithSigningSecret(p.SigningSecret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize event router: %s\n", err.Error())
		os.Exit(1)
	}
	interactionRouter, err := interactionrouter.New(interactionrouter.WithSigningSecret(p.SigningSecret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize interaction router: %s\n", err.Error())
		os.Exit(1)
	}
	client := slack.New(p.BotToken)

	framework := &framework.Framework{
		EventRouter:       eventRouter,
		InteractionRouter: interactionRouter,
		Client:            client,
	}
	framework.Use(
		echo.NewHandler(),
	)

	http.Handle("/slack/events", framework.EventRouter)
	http.Handle("/slack/actions", framework.InteractionRouter)
	http.Handle("/healthz", healthHandler)
	if err := http.ListenAndServe(p.BindAddr, nil); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %s\n", err.Error())
		os.Exit(1)
	}
}

var healthHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})
