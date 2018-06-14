package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/blinsay/lasagnad"
	"github.com/blinsay/lasagnad/commands"
	"github.com/nlopes/slack"
	"github.com/rakyll/globalconf"
	"github.com/sirupsen/logrus"
)

// Flags are parsed using globalconfig
//
// Flags are organized into FlagSets so that anyone who wants to use a config
// file for some (or all) of the options listed here has some structure to
// help them organize.

var (
	authOpts  = flagset("auth", flag.ExitOnError)
	authToken = authOpts.String("token", "", "the auth token to use to connect to Slack")
)

var (
	debugOpts             = flagset("debug", flag.ExitOnError)
	debug                 = flag.Bool("debug", false, "debug mode")
	dumpWebsocketMessages = flag.Bool("dump-websocket-messages", false, "print all received websocket messages to stderr")
)

func main() {
	conf, err := globalconf.New("lasagnad")
	if err != nil {
		log.Fatalf("uh oh, couldn't read config: %s", err)
	}
	conf.EnvPrefix = "GARF_"
	conf.ParseAll()

	slackAPI := slackClient(*authToken, *dumpWebsocketMessages)

	b := &bot{
		Name:           "lasagnad",
		MessageTimeout: 2 * time.Second,
		Logger:         logger(*debug),
		Slack:          slackAPI,
	}

	cmd := lasagnad.NewCommands('!', b.Name, slackAPI)
	cmd.CmdFunc("echo", "echo it back", commands.Echo)
	cmd.CmdFunc("frog", "CLICK ON FROG FOR TIP", commands.FrogTip)

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

func flagset(name string, errorHandling flag.ErrorHandling) *flag.FlagSet {
	set := flag.NewFlagSet(name, errorHandling)
	globalconf.Register(set.Name(), set)
	return set
}
