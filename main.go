package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"discord-teamspeak-notifier/discord"
	"discord-teamspeak-notifier/teamspeak"
)

// Variables used for command line parameters
var (
	Token           string
	Guild           string
	tsServerId      int
	tsUsername      string
	tsPassword      string
	tsUrl           string
	tsIgnoreChannel teamspeak.TsIgnoreChannelType
)



func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&Guild, "g", "", "Guild id")
	flag.IntVar(&tsServerId, "tsServerId", 1, "Teamspeak server id")
	flag.StringVar(&tsUsername, "tsUser", "", "Teamspeak server query username")
	flag.StringVar(&tsPassword, "tsPassword", "", "Teamspeak server query password")
	flag.StringVar(&tsUrl, "tsUrl", "", "Teamspeak server query url")
	flag.Var(&tsIgnoreChannel, "tsIgnoreChannel", "Ignores users in this channel id")
	flag.Parse()
}

func main() {
	tc, err := teamspeak.Init(tsServerId, tsUsername, tsPassword, tsUrl, tsIgnoreChannel)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return 
	}

	dg, err := discord.Init(tc, Token, Guild)
	if err != nil {
		fmt.Printf("Error: %s", err)
		return 
	}

	stopWatching := make(chan bool, 1)
	go discord.WatchOnlineUsers(dg, stopWatching)

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	stopWatching <- true

	// Cleanly close down the Discord session.
	dg.Close()

	// Cleanly close down teamspeak session.
	tc.Close()
}


