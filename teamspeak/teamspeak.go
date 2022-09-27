package teamspeak

import (
	"discord-teamspeak-notifier/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/multiplay/go-ts3"
	"golang.org/x/exp/slices"
)

var teamspeakClientIdMapping map[string]string = make(map[string]string, 0)

type TsIgnoreChannelType []string

func (i *TsIgnoreChannelType) String() string {
	return strings.Join(*i, ",")
}

func (i *TsIgnoreChannelType) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var tsIgnoreChannel TsIgnoreChannelType
var teamspeakUserPresence utils.Set = utils.Set{}


func GetTeamspeakUserPresence() utils.Set {
	return teamspeakUserPresence
}

func Init(tsServerId int, tsUsername string, tsPassword string, tsUrl string, ignoreChannel TsIgnoreChannelType, stopWatchingChan <-chan bool) (*ts3.Client, error) {
	tsIgnoreChannel = ignoreChannel

	c, err := ts3.NewClient(tsUrl)
	if err != nil {
		return c, fmt.Errorf("Error: %s", err)
	}

	if err := c.Login(tsUsername, tsPassword); err != nil {
		return c, fmt.Errorf("Error: %s", err)
	}

	// Switch to the correct teamspeak server
	c.Use(tsServerId)

	// Register for Server and Channel events
	c.Register(ts3.ServerEvents)
	c.Register(ts3.ChannelEvents)

	// Watch teamspeak events
	go watchTeamspeak(c.Notifications(), stopWatchingChan)

	// Fetch all teamspeak users initially when launching this bot
	err = getAllTeamspeakUsers(c)
	if err != nil {
		return c, fmt.Errorf("Error: %s", err)
	}
	return c, nil
}

func watchTeamspeak(ch <-chan ts3.Notification, stopWatchingChan <-chan bool) {
	for {
		select {
			case message := <-ch:
				handleTeamspeakEvent(message)
			case <-stopWatchingChan:
				break
		}
	}
}

func handleTeamspeakEvent(message ts3.Notification) {
	data := message.Data
	// Skip users with type 1 (query connections)
	if data["client_type"] == "1" {
		return
	}

	clientId := data["clid"]

	// Reasonid 0 == connected
	// others are "disconnect" related, voluntary or not.
	if data["reasonid"] == "0" {
		dbId, found := data["client_database_id"]

		// Only new user connection events contain the client database id
		// so when receiving an event about a user switching channels this only contains the current client id
		// Via the teamspeakClientIdMapping we can lookup the current clientId's database id and use that.
		// Else if it is present we store it in the mapping.
		if !found {
			dbId, found = teamspeakClientIdMapping[clientId]
			if !found {
				fmt.Printf("User database id not found! ClientId: %s", clientId)
				return
			}
		} else {
			// Disconnect events don't include any database id, so we need to keep a mapping between current client ids and
			// their database ids.
			teamspeakClientIdMapping[clientId] = dbId
		}


		// If user moves to a channel we want to ignore, ignore the user.
		if slices.Contains(tsIgnoreChannel, data["ctid"]) {
			teamspeakUserPresence.Remove(teamspeakClientIdMapping[clientId])
			return
		}

		// Else add it as a present user
		teamspeakUserPresence.Add(dbId)
		return
	}

	// When a client disconnects we lookup the database id of this client id
	// and remove it as a present user.
	teamspeakUserPresence.Remove(teamspeakClientIdMapping[clientId])
	delete(teamspeakClientIdMapping, clientId)
}

func getAllTeamspeakUsers(c *ts3.Client) error {
	clients, err := c.Server.ClientList()

	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}

	for _, tsUser := range clients {
		// If user is a query client, skip it
		if tsUser.Type == 1 {
			continue
		}

		dbId := strconv.Itoa(tsUser.DatabaseID)

		// If user is in a channel we don't want to ignore, ignore the user.
		if !slices.Contains(tsIgnoreChannel, strconv.Itoa(tsUser.ChannelID)) {
			// Add  currently connected users as present
			teamspeakUserPresence.Add(dbId)
		}

		// Disconnect events don't include any database id, so we need to keep a mapping between current client ids and
		// their database ids.
		userId := strconv.Itoa(tsUser.ID)
		teamspeakClientIdMapping[userId] = dbId
	}
	return nil
}

func GetTeamspeakUserIdByName(c *ts3.Client, name string) (int, error) {
	clients, err := c.Server.ClientList()

	if err != nil {
		return 0, fmt.Errorf("Error: %s", err)
	}

	for _, tsUser := range clients {
		// If user is a query client, skip it
		if tsUser.Type == 1 {
			continue
		}
		// If the user matches the given name, return the database id
		if tsUser.Nickname == name {
			return tsUser.DatabaseID, nil
		}
	}
	return 0, fmt.Errorf("User not found")
}