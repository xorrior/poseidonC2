package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/xorrior/poseidonC2/pkg/servers"
)

var cf *os.File
var wsCF servers.WsConfig
var slackCF servers.SlackC2
var proto string

func main() {
	configFile := flag.String("config", "", "Local file path to the json configuration file. Use this file to set the C2 profile options")
	websocketsProfile := flag.Bool("websockets", false, "-websockets")
	slackProfile := flag.Bool("slack", false, "-slack")

	flag.Parse()
	// Read in the config file
	if len(*configFile) > 0 {
		cf, _ = os.Open(*configFile)
	} else {
		flag.Usage()
		os.Exit(1)
	}

	if *websocketsProfile == true && *slackProfile == false {
		config, _ := ioutil.ReadAll(cf)
		err := json.Unmarshal(config, &wsCF)

		if err != nil {
			log.Println("Failed to marshal configuration file. Are you sure the configuration file is correct?")
			os.Exit(-1)
		}

		ws := servers.WebsocketC2{}.NewServer()
		ws.Run(wsCF)
	}
}
