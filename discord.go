package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slices"
)


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

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	tempUsername, found := teamspeakTempUsernames[m.Author.ID]
	ch, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Something went wrong sending a DM to "+m.Author.Username)
		return
	}

	// If a temp username has not been set yet we need to generate one and notify the user.
	if found == false {
		tmpUsername := randomString()
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
	s.RequestGuildMembers(Guild, "", 100, "members", true)
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