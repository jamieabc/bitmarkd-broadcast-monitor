package messengers

import (
	"github.com/jamieabc/bitmarkd-broadcast-monitor/fault"
	"github.com/nlopes/slack"
)

type slackMessenger struct {
	channelID string
	client    *slack.Client
	rtm       *slack.RTM
}

// Send - send message to slack
func (s *slackMessenger) Send(args ...interface{}) error {
	if 1 > len(args) {
		return fault.InsufficientSlackSendParameter
	}

	message := args[0].(string)
	s.rtm.SendMessage(s.rtm.NewOutgoingMessage(message, s.channelID))
	return nil
}

func NewSlack(token string, channelID string) Messenger {
	client := slack.New(token)
	return &slackMessenger{
		channelID: channelID,
		client:    client,
		rtm:       client.NewRTM(),
	}
}
