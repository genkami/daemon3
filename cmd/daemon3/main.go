package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	eventrouter "github.com/genkami/go-slack-event-router"
	"github.com/genkami/go-slack-event-router/interactionrouter"
	"github.com/slack-go/slack"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"

	"github.com/genkami/daemon3/pkg/framework"
	"github.com/genkami/daemon3/pkg/handlers/echo"
)

type Params struct {
	BindAddr               string
	SlackSecretVersionName string
	Slack                  SlackSecret
}

type SlackSecret struct {
	SigningSecret string `json:"signingSecret"`
	BotToken      string `json:"botToken"`
}

func (p *Params) Setup() {
	flag.StringVar(&p.BindAddr, "addr", "", "bind address")
	flag.StringVar(&p.SlackSecretVersionName, "slack-secret-version-name", "", "the name of the secret version of the Slack secrets")
	flag.Parse()

	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize secret manager client: %v\n", err.Error())
		os.Exit(1)
	}

	resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: p.SlackSecretVersionName,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get signing secret: %s\n", err.Error())
		os.Exit(1)
	}
	err = json.Unmarshal(resp.Payload.Data, &p.Slack)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse slack secrets: %s\n", err.Error())
		os.Exit(1)
	}
}

func main() {
	var p Params
	p.Setup()
	eventRouter, err := eventrouter.New(eventrouter.WithSigningSecret(p.Slack.SigningSecret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize event router: %s\n", err.Error())
		os.Exit(1)
	}
	interactionRouter, err := interactionrouter.New(interactionrouter.WithSigningSecret(p.Slack.SigningSecret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize interaction router: %s\n", err.Error())
		os.Exit(1)
	}
	client := slack.New(p.Slack.BotToken)

	f := &framework.Framework{
		EventRouter:       eventRouter,
		InteractionRouter: interactionRouter,
		Client:            client,
	}
	err = f.Use(
		echo.NewHandler(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set up handlers: %s\n", err.Error())
		os.Exit(1)
	}

	http.Handle("/slack/events", f.EventRouter)
	http.Handle("/slack/actions", f.InteractionRouter)
	http.Handle("/healthz", healthHandler)
	if err := http.ListenAndServe(p.BindAddr, nil); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %s\n", err.Error())
		os.Exit(1)
	}
}

var healthHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})
