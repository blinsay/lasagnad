package commands

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
)

// Echo is a CommandFunc that echoes text back to the user.
func Echo(ctx context.Context, text string, m *slack.MessageEvent) (string, error) {
	return text, nil
}

const (
	frogURL string = "https://frog.tips/api/1/tips/"
)

// FrogTip does a frogtips.
func FrogTip(ctx context.Context, text string, m *slack.MessageEvent) (string, error) {
	request, err := http.NewRequest(http.MethodGet, frogURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "frogtips: http.NewRequest")
	}

	request = request.WithContext(ctx)
	client := http.Client{}

	resp, err := client.Do(request)
	if err != nil {
		return "", errors.Wrap(err, "frogtips: http")
	}

	var tips struct {
		Tips []struct {
			Number float64
			Tip    string
		}
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&tips); err != nil {
		return "", errors.Wrap(err, "frogtips: json")
	}

	return tips.Tips[rand.Intn(len(tips.Tips))].Tip, nil
}
