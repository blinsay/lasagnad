package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/blinsay/lasagnad"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
)

// TODO(???): start using a config file for secrets
// TODO(???): add more commands and plugins and stuff

var (
	debug      = false
	slackDebug = false
	botAuth    = os.Getenv("GARF_BOT_AUTH")
)

func init() {
	flag.BoolVar(&debug, "debug", false, "turn on debug mode")
	flag.BoolVar(&slackDebug, "slack-debug", false, "turn on slack-api debug mode")
	flag.Parse()
}

func main() {
	slackApi := slackClient(botAuth, slackDebug)

	b := &bot{
		Name:           "lasagnad",
		MessageTimeout: 2 * time.Second,
		Logger:         logger(debug),
		Slack:          slackApi,
	}

	cmd := lasagnad.NewCommands('!', b.Name, slackApi)
	cmd.CmdFunc("echo", "echo your message back. how fun!", func(ctx context.Context, text string, event *slack.MessageEvent) (string, error) {
		return text, nil
	})

	b.Observers = []lasagnad.Observer{
		cmd,
	}

	if err := b.TestAuth(); err != nil {
		b.Logger.Fatal("can't start! failed an auth test: ", err)
		return
	}

	if err := b.Run(); err != nil {
		b.Logger.Error("exiting with a fatal error: ", err)
	}
}

func slackClient(botToken string, debug bool) *slack.Client {
	api := slack.New(botToken)
	if debug {
		debugLogger := log.New(os.Stderr, "lasagnadebug: ", log.Lshortfile|log.LUTC|log.LstdFlags)
		slack.SetLogger(debugLogger)
		api.SetDebug(true)
	}
	return api
}

func logger(debug bool) *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.TextFormatter{
		DisableColors:  true,
		FullTimestamp:  true,
		DisableSorting: !debug,
	}

	if debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
	}

	return logger
}
