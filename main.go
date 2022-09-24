package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/multiplay/go-ts3"
	"golang.org/x/exp/slices"
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

func watchOnlineUsers(dg *discordgo.Session, stopWatchingChan <-chan bool) {
	for {
		dg.RequestGuildMembers(Guild, "", 100, "members", true)
		select {
		case <-stopWatchingChan:
			break
		case <-time.After(5 * time.Minute):
			continue
		}
	}

}

func createTempUsername() string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321"
	b := make([]byte, 20)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset)-1)]
	}
	return string(b)
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	tempUsername, found := teamspeakTempUsernames[m.Author.ID]
	ch, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Something went wrong sending a DM to "+m.Author.Username)
		return
	}

	// If a temp username has not been set yet we need to generate one and notify the user.
	if found == false {
		tmpUsername := createTempUsername()
		teamspeakTempUsernames[m.Author.ID] = tmpUsername
		s.ChannelMessageSend(ch.ID, "Please adjust your username on teamspeak to: "+tmpUsername)
		s.ChannelMessageSend(ch.ID, "When done please type !enable_mention again")
		return
	}

	// If a temp username has been found we should check if the username is in use on teamspeak
	tsUserId, found := getTeamspeakUserIdByName(tempUsername)
	if found == false {
		s.ChannelMessageSend(m.ChannelID, "Could not find a user with username: "+tempUsername)
		return
	}

	// If a user has been found on teamspeak, grab its ID and add it to the discordTeamspeak mapping.
	discordTeamspeakMapping[m.Author.ID] = strconv.Itoa(tsUserId)
	delete(teamspeakTempUsernames, m.Author.ID)
	jsonStr, err := json.Marshal(discordTeamspeakMapping)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
	} else {
		_ = ioutil.WriteFile("discordTeamspeakMapping.json", jsonStr, 0644)
	}
	s.ChannelMessageSend(m.ChannelID, "Bot has been enabled, you are now free to change back your name on teamspeak")
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!enable_mention") {
		handleCommand(s, m)
		return
	}

	var message = ""

	for _, userId := range userPresence {
		if userId != m.Author.ID {
			message = message + "<@" + userId + ">"
		}
	}

	if message != "" {
		sentMessage, _ := s.ChannelMessageSend(m.ChannelID, message)
		s.ChannelMessageDelete(m.ChannelID, sentMessage.ID)
	}

}

func presenceHandler(s *discordgo.Session, u *discordgo.GuildMembersChunk) {
	getTeamspeakUsers()
	tmpPresence := make([]string, 0)
	for _, p := range u.Presences {
		if p.User.ID == s.State.User.ID {
			continue
		}

		userTeamspeakId, found := discordTeamspeakMapping[p.User.ID]
		if found == true && slices.Contains(teamspeakUsers, userTeamspeakId) {
			tmpPresence = append(tmpPresence, p.User.ID)
		}
	}
	userPresence = tmpPresence
}

func getTeamspeakUsers() {
	c, err := ts3.NewClient(tsUrl)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	if err := c.Login(tsUsername, tsPassword); err != nil {
		fmt.Println(err)
	}

	if v, err := c.Version(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("server is running:", v)
	}

	c.Use(tsServerId)

	clients, err := c.Server.ClientList()

	teamspeakUsers = make([]string, 0)
	for _, tsUser := range clients {
		// If user is a query client, skip it
		if tsUser.Type == 1 {
			continue
		}
		// If user is in a channel we want to ignore, ignore the user.
		if slices.Contains(tsIgnoreChannel, strconv.Itoa(tsUser.ChannelID)) {
			continue
		}
		teamspeakUsers = append(teamspeakUsers, strconv.Itoa(tsUser.DatabaseID))
	}
}

func getTeamspeakUserIdByName(name string) (int, bool) {
	c, err := ts3.NewClient(tsUrl)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	if err := c.Login(tsUsername, tsPassword); err != nil {
		fmt.Println(err)
	}

	if v, err := c.Version(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("server is running:", v)
	}

	c.Use(tsServerId)

	clients, err := c.Server.ClientList()

	teamspeakUsers = make([]string, 0)
	for _, tsUser := range clients {
		// If user is a query client, skip it
		if tsUser.Type == 1 {
			continue
		}
		if tsUser.Nickname == name {
			return tsUser.DatabaseID, true
		}
	}
	return 0, false
}
