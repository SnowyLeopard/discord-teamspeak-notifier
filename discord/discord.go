package discord

import (
	"discord-teamspeak-notifier/teamspeak"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/multiplay/go-ts3"
	"golang.org/x/exp/slices"
)
var teamspeakTempUsernames map[string]string
var discordTeamspeakMapping map[string]string

var userPresence []string

var (
	Guild string
	TsClient *ts3.Client
)

func Init(tc *ts3.Client, token string, guild string) (*discordgo.Session, error) {
	TsClient = tc
	Guild = guild

	teamspeakTempUsernames = make(map[string]string)

	// Read existing mappings from file
	discordTeamspeakMapping = make(map[string]string)
	file, err := ioutil.ReadFile("discordTeamspeakMapping.json")
	if err == nil {
		json.Unmarshal(file, &discordTeamspeakMapping)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return dg, fmt.Errorf("error creating Discord session: %s", err)
	}

	dg.AddHandler(onMessage)
	dg.AddHandler(onGuildMembers)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers | discordgo.IntentDirectMessages

	err = dg.Open()

	if err != nil {
		return dg, fmt.Errorf("error opening connection: %s", err)
	}

	return dg, err
}

func requestGuildMembers(dg *discordgo.Session) {
	// Request all members for a specific guild (server).
	// Given a query, limit (how much users we want to fetch, 0 means all of them),
	// a "nonce" string, and whether we would like "presence" information of the users
	dg.RequestGuildMembers(Guild, "", 0, "members", true)
}

func WatchOnlineUsers(dg *discordgo.Session, stopWatchingChan <-chan bool) {
	for {
		requestGuildMembers(dg)
		select {
		case <-stopWatchingChan:
			break
		case <-time.After(5 * time.Minute):
			continue
		}
	}

}

func HandleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	tempUsername, found := teamspeakTempUsernames[m.Author.ID]
	ch, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		fmt.Printf("Error: %s", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Something went wrong sending a DM to %s", m.Author.Username))
		return
	}

	// If a temp username has not been set yet we need to generate one and notify the user.
	if found == false {
		tmpUsername := randomString()
		teamspeakTempUsernames[m.Author.ID] = tmpUsername
		s.ChannelMessageSend(ch.ID, fmt.Sprintf("Please adjust your username on teamspeak to: %s", tmpUsername))
		s.ChannelMessageSend(ch.ID, "When done please type !enable_mention again")
		return
	}

	// If a temp username has been found we should check if the username is in use on teamspeak
	tsUserId, err := teamspeak.GetTeamspeakUserIdByName(TsClient, tempUsername)
	if err != nil {
		fmt.Printf("Error: %s", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Could not find a user with username: %s", tempUsername))
		return
	}

	// If a user has been found on teamspeak, grab its ID and add it to the discordTeamspeak mapping.
	discordTeamspeakMapping[m.Author.ID] = strconv.Itoa(tsUserId)
	delete(teamspeakTempUsernames, m.Author.ID)
	jsonStr, err := json.Marshal(discordTeamspeakMapping)
	if err != nil {
		fmt.Printf("Error: %s", err)
		s.ChannelMessageSend(m.ChannelID, "Something went wrong, please contact the developer of this bot.")
		return
	}

	err = ioutil.WriteFile("discordTeamspeakMapping.json", jsonStr, 0644)
	if err != nil {
		fmt.Printf("Error: %s", err)
		s.ChannelMessageSend(m.ChannelID, "Something went wrong, please contact the developer of this bot.")
		return
	}

	requestGuildMembers(s)
	s.ChannelMessageSend(m.ChannelID, "Bot has been enabled, you are now free to change back your name on teamspeak.")
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!enable_mention") {
		HandleCommand(s, m)
		return
	}

	var message = ""

	for _, userId := range userPresence {
		if userId != m.Author.ID {
			message = message + fmt.Sprintf("<@%s>", userId)
		}
	}

	// When message is not empty, and thus contains users we should mention, send the message to the channel.
	// Invoking mentions to the applicable users.
	// We also directly remove the message since the message itself is not relevant, we only want to trigger the mention notification.
	if message != "" {
		sentMessage, _ := s.ChannelMessageSend(m.ChannelID, message)
		s.ChannelMessageDelete(m.ChannelID, sentMessage.ID)
	}

}

func onGuildMembers(s *discordgo.Session, u *discordgo.GuildMembersChunk) {
	teamspeakUsers, err := teamspeak.GetTeamspeakUsers(TsClient)
	if err != nil {
		fmt.Printf("Error %s", err)
		return
	}
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