// +build slack

package servers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/xorrior/poseidon/pkg/utils/structs"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/nlopes/slack"
)

//Slack - Struct definition for slack c2
type SlackC2 struct {
	Key          string
	ChannelID    string
	PollInterval int
	BaseURL      string
	LogFile      string
	ApiHandle 	*slack.Client
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

func (s SlackC2) GetApiHandle() *slack.Client {
	return s.ApiHandle
}

func (s *SlackC2) SetApiHandle(newClient *slack.Client) {
	s.ApiHandle = newClient
}

func (s SlackC2) GetNextTask(apfellID string) []byte {
	//place holder
	url := fmt.Sprintf("%sapi/v1.3/tasks/callback/%s/nextTask", s.ApfellBaseURL(), apfellID)
	return s.htmlGetData(url)
}

func (s SlackC2) PostResponse(taskid string, output []byte) []byte {
	urlEnding := fmt.Sprintf("api/v1.3/responses/%s", taskid)
	return s.postRESTResponse(urlEnding, output)
}

//postRESTResponse - Wrapper to post task responses through the Apfell rest API
func (s *SlackC2) postRESTResponse(urlEnding string, data []byte) []byte {
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
func (s *SlackC2) htmlPostData(urlEnding string, sendData []byte) []byte {
	url := fmt.Sprintf("%s%s", s.ApfellBaseURL(), urlEnding)
	s.GetApiHandle().Debugln(fmt.Sprintln("Sending POST request to: ", url))

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(sendData))
	contentLength := len(sendData)
	req.ContentLength = int64(contentLength)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		s.GetApiHandle().Debugln(fmt.Sprintf("Error sending POST request: %s", err.Error()))
		return make([]byte, 0)
	}

	if resp.StatusCode != 200 {
		s.GetApiHandle().Debugln(fmt.Sprintf("Did not receive 200 response code: %d", resp.StatusCode))
		return make([]byte, 0)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		s.GetApiHandle().Debugln(fmt.Sprintf("Error reading response body: %s", err.Error()))
		return make([]byte, 0)
	}

	return body
}

//htmlGetData - HTTP GET request for data
func (s *SlackC2) htmlGetData(url string) []byte {
	s.GetApiHandle().Debugln("Sending HTML GET request to url: ", url)
	client := &http.Client{}
	var respBody []byte

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.GetApiHandle().Debugln(fmt.Sprintf("Error creating http request: %s", err.Error()))
		return make([]byte, 0)
	}

	resp, err := client.Do(req)

	if err != nil {
		s.GetApiHandle().Debugln(fmt.Sprintf("Error completing GET request: %s", err.Error()))
		return make([]byte, 0)
	}

	if resp.StatusCode != 200 {
		s.GetApiHandle().Debugln(fmt.Sprintf("Did not receive 200 response code: %d", resp.StatusCode))
		return make([]byte, 0)
	}

	defer resp.Body.Close()

	respBody, _ = ioutil.ReadAll(resp.Body)

	return respBody

}

func (s *SlackC2) handleResponseMessage(timestamp string, data []byte) {
	if len(data) < 4000 {
		// If less than 4k bytes, send a normal message

		_, _, _, err := s.GetApiHandle().SendMessage(s.GetChannelID(), slack.MsgOptionTS(timestamp), slack.MsgOptionText(string(data), true))
		if err != nil {
			s.GetApiHandle().Debugf("Unable to post message: %s", err.Error())
			return
		}

	} else if len(data) > 4000 && len(data) < 8000 {
		// Send an attachment
		attachment := slack.Attachment{
			Color:         "",
			Fallback:      "",
			CallbackID:    "",
			ID:            0,
			AuthorID:      "",
			AuthorName:    "",
			AuthorSubname: "",
			AuthorLink:    "",
			AuthorIcon:    "",
			Title:         "",
			TitleLink:     "",
			Pretext:       "",
			Text:          string(data),
			ImageURL:      "",
			ThumbURL:      "",
			Fields:        nil,
			Actions:       nil,
			MarkdownIn:    nil,
			Footer:        "",
			FooterIcon:    "",
			Ts:            "",
		}

		_, _, _, err := s.GetApiHandle().SendMessage(s.GetChannelID(), slack.MsgOptionTS(timestamp), slack.MsgOptionText("",true), slack.MsgOptionAttachments(attachment))
		if err != nil {
			s.GetApiHandle().Debugf("Unable to post message: %s", err.Error())
			return
		}
	} else {
		// Data larger than 8K will be uploaded as a file
		params := slack.FileUploadParameters{
			File:            "newmessage.json",
			Content:         string(data),
			Reader:          nil,
			Filetype:        "",
			Filename:        "newmessage",
			Title:           "",
			InitialComment:  "",
			Channels:        []string{s.GetChannelID()},
			ThreadTimestamp: timestamp,
		}

		_, err := s.GetApiHandle().UploadFile(params)
		if err != nil {
			s.GetApiHandle().Debugf("Unable to post message: %s", err.Error())
			return
		}
	}
}

func (s *SlackC2) handleMessage(message interface{}) interface{} {
	m := message.(Message)

	switch m.MType {
	case CheckInMsg:
		s.GetApiHandle().Debugln("Received CheckIn message")
		resp := s.htmlPostData("api/v1.3/callbacks/", []byte(m.Data))

		// Create the msg to respond to the client
		re := Message{}
		re.Data = string(resp)
		re.MType = CheckInMsg

		return re

	case EKE:
		s.GetApiHandle().Debugf("Received EKE message with IDType: %d", m.IDType)
		s.GetApiHandle().Debugf("Message: %+v\n", m)
		resp := s.htmlPostData(fmt.Sprintf("api/v1.3/crypto/EKE/%s", m.ID), []byte(m.Data))

		if len(resp) == 0 {
			s.GetApiHandle().Debugln("Empty response received from apfell. ...")
			break
		}

		re := Message{}
		re.Data = string(resp)
		re.MType = EKE

		return re
	case AES:
		s.GetApiHandle().Debugln("Received AES message")
		s.GetApiHandle().Debugf("Message: %+v\n", m)
		resp := s.htmlPostData(fmt.Sprintf("api/v1.3/crypto/aes_psk/%s", m.ID), []byte(m.Data))
		if len(resp) == 0 {
			s.GetApiHandle().Debugf("Received empty response from apfell")
			break
		}

		re := Message{}
		re.Data = string(resp)
		re.MType = AES

		return re

	case TaskMsg:
		s.GetApiHandle().Debugln("Received task message")
		s.GetApiHandle().Debugf("Message: %+v\n", m)
		resp := s.GetNextTask(m.ID)

		if len(resp) == 0 {
			s.GetApiHandle().Debugln("Received empty response from Apfell.... retrying ")
			time.Sleep(time.Duration(s.PollingInterval()) * time.Second)

			resp = s.GetNextTask(m.ID)
		}

		re := Message{}
		re.Data = string(resp)
		re.MType = TaskMsg

		return re

	case ResponseMsg:
		s.GetApiHandle().Debugln("Received Task response msg")
		s.GetApiHandle().Debugf("Message: %+v\n", m)
		resp := s.htmlPostData(fmt.Sprintf("api/v1.3/responses/%s", m.ID), []byte(m.Data))

		re := Message{}
		re.Data = string(resp)
		re.MType = ResponseMsg

		return re

	case FileMsg:
		s.GetApiHandle().Debugln("Received file msg")
		s.GetApiHandle().Debugf("Message: %+v\n", m)
		endpoint := fmt.Sprintf("api/v1.3/files/%s/callbacks/%s", m.ID, m.Tag)
		url := fmt.Sprintf("%s%s", s.ApfellBaseURL(), endpoint)
		resp := s.htmlGetData(url)

		re := Message{}
		re.Data = string(resp)
		re.MType = FileMsg

		return re
	}

	return Message{}
}

func (s SlackC2) Run(config interface{}) {

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

	s.SetApiHandle(slack.New(s.GetKey(), slack.OptionDebug(s.Debug), slack.OptionLog(logger)))
	// Join the channel
	/*_, err := api.JoinChannel(s.GetChannelID())

	if err != nil {
		s.GetApiHandle().Debugf("Unable to join channel: %s, err: %s", s.GetChannelID(), err.Error())
		os.Exit(-1)
	}*/

	rtm := s.GetApiHandle().NewRTM()
	go rtm.ManageConnection()

	// Listen for messages from clients

	for msg := range rtm.IncomingEvents {
		s.GetApiHandle().Debugln("New event received ...")

		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Bye felicia
			break
		case *slack.ConnectedEvent:
			s.GetApiHandle().Debugln("Connected to workspace...")
			//s.GetApiHandle().PostMessage(s.GetChannelID(), slack.MsgOptionText("poseidon-online", false))
			break
		case *slack.MessageEvent:
			//s.GetApiHandle().Debugf("Received new message event: %+v\n", ev)

			if len(ev.Text) != 0 && len(ev.Attachments) == 0 && len(ev.Files) == 0 {
				// Normal text message with no attachments and no files
				// Save the timestamp for the tag
				ts := ev.Timestamp
				msg := Message{}

				// Unmarshal the text to a Message struct
				err := json.Unmarshal([]byte(ev.Text), &msg)

				if err != nil {
					s.GetApiHandle().Debugf("Error unmarshaling data from message text: %s", err.Error())
					break
				}

				if msg.Client {

					resp := s.handleMessage(msg)

					respMsg := resp.(Message)
					respMsg.Client = false
					rawResp, err := json.Marshal(respMsg)

					if err != nil {
						s.GetApiHandle().Debugf("Error marshaling data from response message: %s", err.Error())
						break
					}

					go s.handleResponseMessage(ts, rawResp)
				}


			} else if len(ev.Attachments) > 0 {
				// Grab the first attachment and save the time stamp
				att := ev.Attachments[0]
				ts := ev.Timestamp

				msg := Message{}

				err := json.Unmarshal([]byte(att.Text), &msg)
				if err != nil {
					s.GetApiHandle().Debugf("Error unmarshaling data from message text: %s", err.Error())
					break
				}

				if msg.Client {

					resp := s.handleMessage(msg)

					respMsg := resp.(Message)
					respMsg.Client = false
					rawResp, err := json.Marshal(respMsg)

					if err != nil {
						s.GetApiHandle().Debugf("Error marshaling data from response message: %s", err.Error())
						break
					}

					go s.handleResponseMessage(ts, rawResp)
				}
			} else if len(ev.Files) > 0 {
				// Grab the first file and save the time stamp
				slackFile := ev.Files[0]
				ts := ev.Timestamp

				var fileContents bytes.Buffer
				err := s.GetApiHandle().GetFile(slackFile.URLPrivateDownload, &fileContents)

				if err != nil {
					s.GetApiHandle().Debugf("Unable to retrieve file: %s ", err.Error())
					break
				}
				msg := Message{}
				err = json.Unmarshal(fileContents.Bytes(), &msg)

				if err != nil {
					s.GetApiHandle().Debugf("Error unmarshaling data from message text: %s", err.Error())
					break
				}

				if msg.Client {

					resp := s.handleMessage(msg)

					respMsg := resp.(Message)
					respMsg.Client = false
					rawResp, err := json.Marshal(respMsg)

					if err != nil {
						s.GetApiHandle().Debugf("Error marshaling data from response message: %s", err.Error())
						break
					}

					go s.handleResponseMessage(ts, rawResp)
				}
			}

			break

		}
	}

}

func (s SlackC2) ConvertEncoding(old string) string {
	decoded, _ := base64.URLEncoding.DecodeString(old)
	return base64.StdEncoding.EncodeToString(decoded)
}
