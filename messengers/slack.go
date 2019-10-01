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

// Valid - check if slack instance successful created
func (s *slackMessenger) Valid() bool {
	return nil != s.client
}

// NewSlack - create slack client
func NewSlack(token string, channelID string) Messenger {
	if "" != token && "" != channelID {
		client := slack.New(token)
		rtm := client.NewRTM()
		go rtm.ManageConnection()
		return &slackMessenger{
			channelID: channelID,
			client:    client,
			rtm:       rtm,
		}
	}
	return &slackMessenger{}
}
