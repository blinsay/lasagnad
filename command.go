package lasagnad

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

// TODO(benl): allow commands to stream responses somehow
// TODO(benl): handle panics
// TODO(benl): subcommands (like `!img pin garf` and `!img show garf` or `!db version` and `!db update`)

// A Command is a high-level interface to the slack API.
//
// Commands are run in response to matching a command string and respond to the
// triggering message.
type Command interface {
	// Help returns a short description of this command.
	Help() string

	// RunCommand executes a command. The returned reply string is posted in
	// response to the triggering message. If the reply string is empty or an
	// error is returned, no reply is sent.
	//
	// RunCommand is passed the message text and the complete message that
	// triggered it. Callers may modify the message text before calling this
	// method.
	RunCommand(ctx context.Context, text string, msg *slack.MessageEvent) (reply string, err error)
}

// a commandFunc is a wrapper for a func and some help
type commandFunc struct {
	help string
	run  func(context.Context, string, *slack.MessageEvent) (string, error)
}

func (c *commandFunc) Help() string {
	return c.help
}

func (c *commandFunc) RunCommand(ctx context.Context, text string, msg *slack.MessageEvent) (string, error) {
	return c.run(ctx, text, msg)
}

// Commands registers multiple commands to run in response to seeing a message
// starting with a prefix and the command name. For example, Commands may use
// "!" as a prefix - in that case "!help" and "!echo" would run the "help" and
// "echo" commands.
//
// Commands is an Observer that muxes multiple Command instances.
type Commands struct {
	username string
	client   *slack.Client
	re       *regexp.Regexp
	cmds     map[string]Command
}

// NewCommands creates a new command muxer. The muxer will match any messages
// that start with the given rune, and ignore any messages sent by it's own
// username.
func NewCommands(prefix rune, username string, client *slack.Client) *Commands {
	cmds := &Commands{
		username: username,
		client:   client,
		re:       regexp.MustCompile(fmt.Sprintf(`^%c(\w+)\s*`, prefix)),
		cmds:     make(map[string]Command),
	}

	cmds.CmdFunc("help", "list all commands", cmds.help)
	return cmds
}

func (c *Commands) help(context.Context, string, *slack.MessageEvent) (string, error) {
	var helps []string
	for name, cmd := range c.cmds {
		helps = append(helps, fmt.Sprintf("!%s - %s", name, cmd.Help()))
	}
	return strings.Join(helps, "\n"), nil
}

// Cmd registers a command with the given name. Cmd panics if multiple Commands
// are registered with the same name.
func (c *Commands) Cmd(name string, cmd Command) {
	if _, found := c.cmds[name]; found {
		// Fucking up routes is a programming error and not a thing we can figure
		// out at compile time. yell and scream and crash the program and hate
		// mondays.
		panic("can't register the same command twice")
	}

	c.cmds[name] = cmd
}

// CmdFunc registers a function as a Command with the given name and help
// string. CmdFunc panics if multiple commands are registered with the same
// name.
func (c *Commands) CmdFunc(name string, help string, f func(context.Context, string, *slack.MessageEvent) (string, error)) {
	c.Cmd(name, &commandFunc{
		help: help,
		run:  f,
	})
}

// Observe muxes over all registered Command implementations for every
// MessageEvent. Discards all other message types.
func (c *Commands) Observe(ctx *BotContext, event *slack.RTMEvent) error {
	e, isEvent := event.Data.(*slack.MessageEvent)
	if !isEvent {
		return nil
	}
	if e.Username == c.username {
		return nil
	}

	log := ctx.Log.WithField("observer", "commands")
	bounds := c.re.FindStringSubmatchIndex(e.Text)
	if bounds == nil {
		log.Debug("message not matched")
		return nil
	}

	cmdName := e.Text[bounds[2]:bounds[3]]
	log = log.WithField("cmd", cmdName)

	cmd, found := c.cmds[cmdName]
	if !found {
		log.Debug("cmd not found")
		return nil
	}
	log.Debug("running command")

	cleanText := e.Text[bounds[1]:]
	replyText, err := cmd.RunCommand(ctx.Ctx, cleanText, e)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("cmd - %s:", cmdName))
	}

	if replyText != "" {
		_, _, err = c.client.PostMessageContext(ctx.Ctx, e.Channel, replyText, slack.PostMessageParameters{})
		if err != nil {
			return errors.Wrap(err, "slack.message.post")
		}
	}

	return nil
}
