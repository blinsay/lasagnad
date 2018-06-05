package main

import (
	"context"
	"fmt"
	"time"

	"github.com/blinsay/lasagnad"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

// TODO(benl): track latency
// TODO(benl): track uptime
// TODO(benl): track messages currently in flight
// TODO(benl): stats command that shows all of the above

type bot struct {
	// the name of the bot, determined when logging on
	Name string
	// the slack user id of this bot, determined when logging on. note that this
	// is NOT the same as the bot_id of this bot!
	UserID string
	// the amount of time the bot is allowed to spend handling a single message.
	MessageTimeout time.Duration

	Observers []lasagnad.Observer
	Slack     *slack.Client
	Logger    logrus.FieldLogger
}

// test auth against slack and validate that Name and UserID are empty or
// match whatever comes back from the Slack API.
//
// this call MUST return before it's safe to call Run
func (b *bot) TestAuth() error {
	resp, err := b.Slack.AuthTest()
	if err != nil {
		return err
	}

	if b.Name == "" {
		b.Name = resp.User
	}
	if resp.User != b.Name {
		return fmt.Errorf("testauth: configured and actual usernames differ: configured=%q actual=%q", b.Name, resp.User)
	}

	if b.UserID == "" {
		b.UserID = resp.UserID
	}
	if resp.UserID != b.UserID {
		return fmt.Errorf("testauth: configured and actual user_id differ: configured=%q actual=%q", b.UserID, resp.UserID)
	}

	return nil
}

// run this bot. any errors returned from Run can be considered fatal and should
// probably terminate the program.
func (b *bot) Run() error {
	rtm := b.Slack.NewRTM()
	go rtm.ManageConnection()

	for message := range rtm.IncomingEvents {
		if event, isErr := message.Data.(error); isErr {
			b.Logger.WithField("error", event).Error("unhandled error")
			return event
		}

		switch event := message.Data.(type) {
		case *slack.ConnectedEvent:
			b.Logger.Info("connected")
		case *slack.LatencyReport:
			b.Logger.WithField("latency", event.Value).Debug("latency report")
		case *slack.Ping:
			b.Logger.WithField("ping_id", event.ID).Debug("ping")
		}

		go b.observe(&message)
	}

	return nil /*unreachable*/
}

// observe the event with every registered observer. observers are called
// sequentially and in the same goroutine for each message - letting a slow
// observer block another is a choice we're gonna stick to for now. if an
// Observer knows it's slow, it can fire off its own goroutines.
func (b *bot) observe(event *slack.RTMEvent) {
	timeout, cancel := context.WithTimeout(context.Background(), b.MessageTimeout)
	defer cancel()

	ctx := lasagnad.BotContext{
		Ctx: timeout,
		Log: b.Logger,
	}

	// TODO(benl): find or generate a unique id for the incoming event
	// TODO(benl): the id to the logger passed into Observers

	for _, observer := range b.Observers {
		if err := observer.Observe(&ctx, event); err != nil {
			b.Logger.WithField("error", err).Error("error observing message")
		}

		if err := timeout.Err(); err != nil {
			b.Logger.WithField("timeout_reason", err).Error("Timed out")
			// NOTE: it would definitely be an ok choice to let observers keep
			// going and operate in some kind of degraded state. do that later if
			// there is actually a reason for it.
			break
		}
	}
}
