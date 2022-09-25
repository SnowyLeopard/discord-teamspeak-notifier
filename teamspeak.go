package main

import (
	"fmt"
	"strconv"

	"github.com/multiplay/go-ts3"
	"golang.org/x/exp/slices"
)

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