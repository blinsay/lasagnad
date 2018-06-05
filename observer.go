package lasagnad

import (
	"context"

	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

// A BotContext is a context object that wraps both a context.Context that an
// Observer should respect and the logger it should use to handle the
// message.
type BotContext struct {
	Ctx context.Context
	Log logrus.FieldLogger
}

// An Observer is a low-level hook into the slack API. Observers are called with
// a BotContext for every message seen using the RTM api.
type Observer interface {
	Observe(ctx *BotContext, event *slack.RTMEvent) error
}
