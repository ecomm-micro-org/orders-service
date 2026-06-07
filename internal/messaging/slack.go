package messaging

import (
	"fmt"

	"github.com/slack-go/slack"
)

type SlackMessenger struct {
	slackClient  *slack.Client
	slackChannel string
}

func NewSlackMessenger(slackClient *slack.Client, slackChannel string) (*SlackMessenger, error) {
	if slackClient == nil {
		return nil, fmt.Errorf("slack client cannot be nil")
	}
	return &SlackMessenger{
		slackClient:  slackClient,
		slackChannel: slackChannel,
	}, nil
}

func (m *SlackMessenger) SendMsg(msg string) error {
	msgOption := slack.MsgOptionText(msg, false)
	options := []slack.MsgOption{msgOption}
	_, _, err := m.slackClient.PostMessage(m.slackChannel, options...)
	return err
}
