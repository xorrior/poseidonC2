package servers

type Server interface {
	PollingInterval() int
	SetPollingInterval(interval int)
	ApfellBaseURL() string
	SetApfellBaseURL(url string)
	PostResponse(taskid int, data []byte)
	GetNextTask(apfellID int) []byte
	SendClientMessage(apfellID int, data []byte)
}
