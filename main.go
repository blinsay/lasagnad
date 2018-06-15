package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/nlopes/slack"
	"github.com/rakyll/globalconf"
	"github.com/sirupsen/logrus"
)

// Flags are parsed using globalconfig
//
// Flags are organized into FlagSets so that anyone who wants to use a config
// file for some (or all) of the options listed here has some structure to
// help them organize.
//
// NOTE: only global Flag opts are parseable from the CLI

// global opts
var (
	debug                 = flag.Bool("debug", false, "debug mode")
	dumpWebsocketMessages = flag.Bool("dump-websocket-messages", false, "print all received websocket messages to stderr")
)

// image options
var (
	imgOpts         = flagset("img")
	imgBucket       = imgOpts.String("bucket", "", "the s3 bucket to store images in")
	imgPrefix       = imgOpts.String("prefix", "", "the s3 prefix to use to namespace images")
	imgMaxSizeBytes = imgOpts.Int64("max-size-bytes", -1, "the maximum allowed image size, in bytes")
)

// auth opts
var (
	authOpts  = flagset("auth")
	authToken = authOpts.String("token", "", "the auth token to use to connect to Slack")
)

func main() {
	conf, err := globalconf.New("lasagnad")
	if err != nil {
		log.Fatalf("uh oh, couldn't read config: %s", err)
	}
	conf.EnvPrefix = "GARF_"
	conf.ParseAll()

	if *imgBucket == "" || *imgPrefix == "" || *imgMaxSizeBytes <= 0 {
		log.Fatalf("invalid s3 bucket config! need a bucket, prefix, and a valid max size in bytes")
	}

	b := &bot{
		Name:           "lasagnad",
		MessageTimeout: 2 * time.Second,
		Logger:         logger(*debug),
		Slack:          slackClient(*authToken, *dumpWebsocketMessages),
		dump: &imgdump{
			S3:     s3.New(session.Must(session.NewSession())),
			Bucket: *imgBucket,
			Prefix: *imgPrefix,
		},
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

func flagset(name string) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	globalconf.Register(set.Name(), set)
	return set
}

// TODO(benl): handle panics
// TODO(benl): track latency
// TODO(benl): track uptime
// TODO(benl): stats
// TODO(benl): better command parsing (and maybe debug info into slack?)

type bot struct {
	// the name of the bot, determined when logging on
	Name string

	// the slack user id of this bot, determined when logging on. that this
	// is NOT the same as the bot_id of this bot.
	UserID string

	// the amount of time the bot is allowed to spend handling a single message.
	MessageTimeout time.Duration

	// the imgdump for storing pinned images
	dump *imgdump

	HTTP   http.Client
	Slack  *slack.Client
	Logger logrus.FieldLogger
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
		log := b.Logger.WithField("request_id", uuid.New())

		if event, isErr := message.Data.(error); isErr {
			log.WithField("error", event).Error("unhandled error")
			return event
		}

		switch event := message.Data.(type) {
		case *slack.ConnectedEvent:
			log.Info("connected")
		case *slack.LatencyReport:
			log.WithField("latency", event.Value).Debug("latency report")
		case *slack.Ping:
			log.WithField("ping_id", event.ID).Debug("ping")
		}

		go b.handle(log, &message)
	}

	return nil /*unreachable*/
}

var (
	commandRe              = regexp.MustCompile(`^!(pin|show|list)\s*(.*)`)
	validPinNameRe         = regexp.MustCompile(`[a-zA-Z0-9]`)
	invalidPinNameResponse = fmt.Sprintf("you made an opps! that's not a valid pin name %s.", validPinNameRe.String())
)

const (
	pinUsage             = "opps! try `!pin LINK NAME` instead."
	showUsage            = "opps, there's nothing to show. try `!show NAME`."
	invalidURLResponse   = "you made an opps! that's not a valid URL."
	pinExists            = "that pin already exists! pins are forever."
	genericErrorResponse = "opps. something went wrong."
)

/// handle every incoming message in a goroutine
func (b *bot) handle(log logrus.FieldLogger, rtmEvent *slack.RTMEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), b.MessageTimeout)
	defer cancel()

	message, isMessage := rtmEvent.Data.(*slack.MessageEvent)
	if !isMessage {
		return
	}

	bounds := commandRe.FindStringSubmatchIndex(message.Text)
	if bounds == nil {
		log.Debug("message not matched")
		return
	}

	// bounds is an array of index pairs that's been flattened. the first pair
	// (0, 1) is the range of the complete match, which will be the entire range
	// of message.Text, the next pair (2, 3) is the range that contains the command,
	// and the last pair (4, 5) is the range of the rest of the message.
	cmd, args := message.Text[bounds[2]:bounds[3]], strings.Fields(message.Text[bounds[4]:])

	// do an early timeout check before trying to do any work
	if err := ctx.Err(); err != nil {
		log.WithField("error", err).Info("timed out")
		return
	}

	startedAt := time.Now()
	defer func() {
		elapsed := time.Since(startedAt)
		log.WithField("elapsed_ms", elapsed/time.Millisecond).Debug("complete")
	}()

	// TODO(benl): catch panics here?
	// TODO(benl): maybe the reply should include something about what happened if
	// an error or panic bubbles up
	switch cmd {
	case `pin`:
		b.handlePin(ctx, log, message, args)
	case `show`:
		b.handleShow(ctx, log, message, args)
	default:
		log.WithField("cmd", cmd).Error("opps. unimplemented command")
		b.reply(ctx, log, message, "opps i don't know that song")
	}
}

func (b *bot) handlePin(ctx context.Context, log logrus.FieldLogger, message *slack.MessageEvent, args []string) {
	if len(args) < 2 {
		b.reply(ctx, log, message, pinUsage)
		return
	}
	urlString, name := args[0], args[1]

	// parse and validate the URL and the pin name. the URL has to be a valid URL
	// and the pin name has to be pretty restricted.
	//
	// note: slack surrounds all URLs with < > so that you can detect them easily.
	// it means that we've gotta strip things here. this is JAAAANKY so maybe there
	// should be a url.ParseSlack func somewhere
	if strings.HasPrefix(urlString, "<") {
		urlString = urlString[1:]
	}
	if strings.HasSuffix(urlString, ">") {
		urlString = urlString[:len(urlString)-1]
	}
	url, err := url.Parse(urlString)
	if err != nil {
		log.WithError(err).Error("invalid url")
		b.reply(ctx, log, message, invalidURLResponse)
		return
	}
	if !validPinNameRe.MatchString(name) {
		log.WithField("pin_name", name).Debug("pin name invalid")
		b.reply(ctx, log, message, invalidPinNameResponse)
		return
	}

	// TODO(benl): give fetch its own timeout, shorter than the total response one. child contexts!
	imageBytes, filetype, err := fetchImageBytes(ctx, &b.HTTP, url, *imgMaxSizeBytes)
	if err != nil {
		log.WithError(err).Error("fetch failed")
		b.reply(ctx, log, message, genericErrorResponse)
		return
	}

	// TODO(benl): give upload its own timeout, shorter than the total response one. child contexts!
	escapedURL := url.String()
	uploader := message.Username
	img, err := b.dump.add(ctx, name, filetype, imageBytes, map[string]*string{
		"uploaded-by":  &uploader,
		"original-url": &escapedURL,
	})

	if err != nil {
		log.WithError(err).Error("upload failed")
		b.reply(ctx, log, message, genericErrorResponse)
		return
	}

	log.WithFields(logrus.Fields{
		"name": img.Name,
		"img":  img.ID,
	}).Info("uploaded")

	b.reply(ctx, log, message, "k")
}

func (b *bot) handleShow(ctx context.Context, log logrus.FieldLogger, message *slack.MessageEvent, args []string) {
	if len(args) < 1 {
		b.reply(ctx, log, message, showUsage)
		return
	}
	name := args[0]

	imgs, err := b.dump.list(ctx, name)
	if err != nil {
		log.WithError(err).Error("listing images failed")
		b.reply(ctx, log, message, genericErrorResponse)
		return
	}

	if len(imgs) == 0 {
		b.reply(ctx, log, message, "there's nothing there :(")
		return
	}

	img := imgs[rand.Intn(len(imgs))]
	b.reply(ctx, log, message, img.URL.String())
}

// reply sends a message back to slack in reponse to something and logs if
// there's an error.
func (b *bot) reply(ctx context.Context, log logrus.FieldLogger, to *slack.MessageEvent, text string) {
	_, _, err := b.Slack.PostMessageContext(ctx, to.Channel, text, slack.PostMessageParameters{
		Markdown:    true,
		UnfurlMedia: true,
	})
	if err != nil {
		log.WithError(err).Error("reply failed")
	}
}
