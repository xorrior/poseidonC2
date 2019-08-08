package servers

const (
	//CheckInMsg - Messages for apfell
	CheckInMsg = 0
	//EKE - Messages for apfell EKE AES
	EKE = 1
	//AES - Messages for apfell static AES
	AES = 2
	//TaskMsg - Messages for apfell tasks
	TaskMsg = 3
	//ResponseMsg - Messages for apfell task responses
	ResponseMsg = 4
	//FileMsg - Messages for apfell file downloads/uploads
	FileMsg = 5
	// ID Type for UUID
	UUIDType = 6
	// ID Type for ApfellID
	ApfellIDType = 7
	// ID Type for FileID
	FileIDType = 8
	// ID Type for session ID
	SESSIDType = 9
	// ID Type for Task ID
	TASKIDType = 10
)

// C2Config - struct for server configuration
type C2Config struct {
	PollInterval int    `json:"pollinterval"`
	BaseURL      string `json:"baseurl"`
	BindAddress  string `json:"bindaddress"`
	SocketURI    string `json:"websocketuri"`
	SlackChannel string `json:"slackchannel"`
	SlackAPIToken string `json:"slackapitoken"`
	SSLKey       string `json:"sslkey"`
	SSLCert      string `json:"sslcert"`
	UseSSL       bool   `json:"usessl"`
	Defaultpage  string `json:"defaultpage"`
	Logfile      string `json:"logfile"`
	Debug        bool   `json:"debug"`
}

//Server - interface used for all c2 profiles
type Server interface {
	PollingInterval() int
	SetPollingInterval(interval int)
	ApfellBaseURL() string
	SetApfellBaseURL(url string)
	PostResponse(taskid string, output []byte) []byte
	GetNextTask(apfellID string) []byte
	SendClientMessage(apfellID string, data []byte)
	Run(cf interface{})
}

//Message - struct definition for messages between clients and the server
type Message struct {
	Tag    string `json:"tag"`
	MType  int    `json:"mtype"`
	IDType int    `json:"idtype"`
	ID     string `json:"id"`
	Data   string `json:"data"`
}

func NewInstance() interface{} {
	return newServer()
}
