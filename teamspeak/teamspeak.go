package teamspeak

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/multiplay/go-ts3"
	"golang.org/x/exp/slices"
)

var teamspeakUsers []string

type TsIgnoreChannelType []string

func (i *TsIgnoreChannelType) String() string {
	return strings.Join(*i, ",")
}

func (i *TsIgnoreChannelType) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var tsIgnoreChannel TsIgnoreChannelType

func Init(tsServerId int, tsUsername string, tsPassword string, tsUrl string, ignoreChannel TsIgnoreChannelType) (*ts3.Client, error) {
	tsIgnoreChannel = ignoreChannel

	c, err := ts3.NewClient(tsUrl)
	if err != nil {
		return c, fmt.Errorf("Error: %s", err)
	}

	if err := c.Login(tsUsername, tsPassword); err != nil {
		return c, fmt.Errorf("Error: %s", err)
	}

	c.Use(tsServerId)
	return c, nil
}

func GetTeamspeakUsers(c *ts3.Client) ([]string, error) {
	clients, err := c.Server.ClientList()
	teamspeakUsers = make([]string, 0)

	if err != nil {
		return teamspeakUsers, fmt.Errorf("Error: %s", err)
	}

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
	return teamspeakUsers, nil
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
		if tsUser.Nickname == name {
			return tsUser.DatabaseID, nil
		}
	}
	return 0, fmt.Errorf("User not found")
}