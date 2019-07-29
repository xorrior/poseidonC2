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

//Message - struct definition for messages between clients and the server
type Message struct {
	Tag    string `json:"tag"`
	MType  int    `json:"mtype"`
	IDType int    `json:"idtype"`
	ID     int    `json:"id"`
	Enc    bool   `json:"enc"`
	Data   string `json:"data"`
}

func (s SlackC2) NewServer() Server {
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

func (s SlackC2) GetNextTask(apfellID int) []byte {
	//place holder
	return make([]byte, 0)
}

func (s SlackC2) PostResponse(taskid int, data []byte) []byte {
	return make([]byte, 0)
}

func (s SlackC2) SendClientMessage(apfellID int, data []byte) {

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

func (s SlackC2) Run() {
	// Set debug and logging options
	f, _ := os.Create("slack.log")
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
