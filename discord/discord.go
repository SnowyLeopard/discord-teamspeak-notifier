package discord

import (
	"discord-teamspeak-notifier/teamspeak"
	"discord-teamspeak-notifier/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/multiplay/go-ts3"
)

var teamspeakTempUsernames map[string]string
var discordTeamspeakMapping map[string]string

var discordUserPresence utils.Set = utils.Set{}

var (
	Guild    string
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

	dg.AddHandler(onMessage)        // Add handler for when any message is received
	dg.AddHandler(onGuildMembers)   // Add handler for the "request guild members" response
	dg.AddHandler(onPresenceUpdate) // Add handler for user presence update events

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers | discordgo.IntentDirectMessages

	err = dg.Open()

	if err != nil {
		return dg, fmt.Errorf("error opening connection: %s", err)
	}

	requestGuildMembers(dg)

	return dg, err
}

func requestGuildMembers(dg *discordgo.Session) {
	// Request all members for a specific guild (server).
	// Given a query, limit (how much users we want to fetch, 0 means all of them),
	// a "nonce" string, and whether we would like "presence" information of the users
	dg.RequestGuildMembers(Guild, "", 0, "members", true)
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
		tmpUsername := utils.RandomString()
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

	s.ChannelMessageSend(m.ChannelID, "Bot has been enabled, you are now free to change back your name on teamspeak.")
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// If message is starting with the enable command, handle it accordingly.
	if strings.HasPrefix(m.Content, "!enable_mention") {
		HandleCommand(s, m)
		return
	}

	// See if any user needs mentioning
	var message = ""
	teamspeakUserPresence := teamspeak.GetTeamspeakUserPresence()

	for userId := range discordUserPresence {
		teamspeakUserId, found := discordTeamspeakMapping[userId]
		// If this userId cannot be matched to any teamspeak user
		// or this discord user is not present on teamspeak
		// or the userId matches the one sending the current message we continue.
		if !found || !teamspeakUserPresence.Has(teamspeakUserId) || userId == m.Author.ID {
			continue
		}

		message = message + fmt.Sprintf("<@%s>", userId)
	}

	// When the message is still empty we don't try sending any messages
	if message == "" {
		return
	}

	// Send the message to the channel.
	// We also directly remove the message since the message itself is not relevant, we only want to trigger the mention notification.
	sentMessage, _ := s.ChannelMessageSend(m.ChannelID, message)
	s.ChannelMessageDelete(m.ChannelID, sentMessage.ID)

}

func onGuildMembers(s *discordgo.Session, u *discordgo.GuildMembersChunk) {
	for _, p := range u.Presences {
		// If the user matches this bot, skip it.
		// If the user is not online, skip it.
		if p.User.ID == s.State.User.ID || p.Status != "online" {
			continue
		}

		discordUserPresence.Add(p.User.ID)
	}
}

func onPresenceUpdate(s *discordgo.Session, u *discordgo.PresenceUpdate) {
	// When the user switches to any status other than "online" we remove the user from the discord presence list
	if u.Presence.Status != "online" {
		discordUserPresence.Remove(u.User.ID)
		return
	}

	// Else we add the user
	discordUserPresence.Add(u.User.ID)
}
