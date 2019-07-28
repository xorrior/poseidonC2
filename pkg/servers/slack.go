package servers

//Slack - Struct definition for slack c2
type Slack struct {
	Key          string
	ChannelID    string
	PollInterval int
	BaseURL      string
}

func (s Slack) NewServer() Server {
	return &Slack{}
}

//PollingInterval - Returns the polling interval
func (s Slack) PollingInterval() int {
	return s.PollInterval
}

//SetPollingInterval - Sets the polling interval
func (s *Slack) SetPollingInterval(interval int) {
	s.PollInterval = interval
}

//ApfellBaseURL - Returns the base url for apfell
func (s Slack) ApfellBaseURL() string {
	return s.BaseURL
}

//SetApfellBaseURL - Sets the base url for apfell
func (s *Slack) SetApfellBaseURL(url string) {
	s.BaseURL = url
}

func (s Slack) GetNextTask(apfellID int) []byte {
	//place holder
	return make([]byte, 0)
}

func (s Slack) PostResponse(taskid int, data []byte) {

}

func (s Slack) SendClientMessage(apfellID int, data []byte) {

}
