package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/namsral/flag"

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
	flag.StringVar(&Token, "discord_bot_token", "", "Bot Token")
	flag.StringVar(&Guild, "discord_guild_id", "", "Guild id")
	flag.IntVar(&tsServerId, "ts_server_id", 1, "Teamspeak server id")
	flag.StringVar(&tsUsername, "ts_user", "", "Teamspeak server query username")
	flag.StringVar(&tsPassword, "ts_password", "", "Teamspeak server query password")
	flag.StringVar(&tsUrl, "ts_url", "", "Teamspeak server query url")
	flag.Var(&tsIgnoreChannel, "ts_ignore_channel", "Ignores users in this channel id")
	flag.Parse()
}

func main() {
	stopWatching := make(chan bool, 1)
	tc, err := teamspeak.Init(tsServerId, tsUsername, tsPassword, tsUrl, tsIgnoreChannel, stopWatching)
	// Cleanly close down teamspeak session.
	defer tc.Close()

	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}

	dg, err := discord.Init(tc, Token, Guild)
	// Cleanly close down the Discord session.
	defer dg.Close()

	if err != nil {
		fmt.Printf("Error: %s", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	stopWatching <- true
}
