package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type tsIgnoreChannelType []string

func (i *tsIgnoreChannelType) String() string {
	return "my string representation"
}

func (i *tsIgnoreChannelType) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Variables used for command line parameters
var (
	Token           string
	Guild           string
	tsServerId      int
	tsUsername      string
	tsPassword      string
	tsUrl           string
	tsIgnoreChannel tsIgnoreChannelType
)

var discordTeamspeakMapping map[string]string
var userPresence []string
var teamspeakUsers []string

var teamspeakTempUsernames map[string]string

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Guild, "g", "", "Guild id")
	flag.IntVar(&tsServerId, "tsServerId", 1, "Teamspeak server id")
	flag.StringVar(&tsUsername, "tsUser", "", "Teamspeak server query username")
	flag.StringVar(&tsPassword, "tsPassword", "", "Teamspeak server query password")
	flag.StringVar(&tsUrl, "tsUrl", "", "Teamspeak server query url")
	flag.Var(&tsIgnoreChannel, "tsIgnoreChannel", "Ignores users in this channel id")
	flag.Parse()

	teamspeakTempUsernames = make(map[string]string)

	// Read existing mappings from file
	discordTeamspeakMapping = make(map[string]string)
	file, err := ioutil.ReadFile("discordTeamspeakMapping.json")
	if err == nil {
		json.Unmarshal(file, &discordTeamspeakMapping)
	}
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(presenceHandler)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers | discordgo.IntentDirectMessages

	err = dg.Open()

	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	stopWatching := make(chan bool, 1)

	go watchOnlineUsers(dg, stopWatching)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	stopWatching <- true

	// Cleanly close down the Discord session.
	dg.Close()
}


