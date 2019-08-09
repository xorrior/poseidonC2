// +build slack

package servers

import (
	"log"
	"os"

	"github.com/nlopes/slack"
)

//Slack - Struct definition for slack c2
type SlackC2 struct {
	Key          string
	ChannelID    string
	PollInterval int
	BaseURL      string
	LogFile      string
	Debug        bool
}

func newServer() Server {
	return &SlackC2{}
}

//PollingInterval - Returns the polling interval
func (s SlackC2) PollingInterval() int {
	return s.PollInterval
}

//SetPollingInterval - Sets the polling interval
func (s *SlackC2) SetPollingInterval(interval int) {
	s.PollInterval = interval
}

//ApfellBaseURL - Returns the base url for apfell
func (s SlackC2) ApfellBaseURL() string {
	return s.BaseURL
}

//SetApfellBaseURL - Sets the base url for apfell
func (s *SlackC2) SetApfellBaseURL(url string) {
	s.BaseURL = url
}

func (s SlackC2) GetNextTask(apfellID string) []byte {
	//place holder
	return make([]byte, 0)
}

func (s SlackC2) PostResponse(taskid string, output []byte) []byte {
	return make([]byte, 0)
}

func (s SlackC2) SendClientMessage(apfellID string, data []byte) {

}

func (s *SlackC2) SetDebug(debug bool) {
	s.Debug = debug
}

func (s *SlackC2) SetLogFile(file string) {
	s.LogFile = file
}

func (s SlackC2) GetKey() string {
	return s.Key
}

func (s *SlackC2) SetKey(newkey string) {
	s.Key = newkey
}

func (s *SlackC2) SetChannelID(ch string) {
	s.ChannelID = ch
}

func (s SlackC2) GetChannelID() string {
	return s.ChannelID
}

func (s *SlackC2) Run(config interface{}) {

	cf := config.(C2Config)
	s.SetKey(cf.SlackAPIToken)
	s.SetChannelID(cf.SlackChannel)
	s.SetApfellBaseURL(cf.BaseURL)
	s.SetDebug(cf.Debug)
	s.SetLogFile(cf.Logfile)
	s.SetPollingInterval(cf.PollInterval)
	// Set debug and logging options
	f, _ := os.Create(s.LogFile)
	logger := log.New(f, "poseidon-slackc2: ", log.Lshortfile|log.LstdFlags)
	api := slack.New(s.GetKey(), slack.OptionDebug(s.Debug), slack.OptionLog(logger))

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	// Listen for messages from clients

	for msg := range rtm.IncomingEvents {
		api.Debugln("New event received ...")

		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Bye felicia

		case *slack.ConnectedEvent:
			api.Debugln("Connected to workspace...")

		case *slack.MessageEvent:
			api.Debugf("Received new message event: %+v\n", ev)
			//TODO: Handle Messages

		}
	}

}
