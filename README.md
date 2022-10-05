# Discord teamspeak notifier

A discord bot that mentions any user that is currently online on discord and on teamspeak.
This has been developed to use a discord chat channel in combination with a teamspeak server since the teamspeak chat functionality is not as feature rich as discord (e.g. you can't paste any images on teamspeak chat).

This bot keeps a list of online users on discord and on teamspeak. When a user is present on both of them the bot will send a message mentioning all applicable users (excluding the user sending the message) after any sent message on discord. This allows any user to set the notification option of the discord channel to "mention only".
The bot will remove the message after sending it to avoid clutter.

## Requirements

- Teamspeak server

  - ServerQuery login

- Discord server

  - Discord application and bot (https://discord.com/developers/applications)

    - The application Oauth2 url should include the following:

      - Scopes

        - bot

      - Bot permissions

        - Send Messages

  - When adding the bot via the discord developer portal you should enable the `Privileged Gateway Intents`

    - Presence intent
    - Server members intent
    - Message content intent

    If you forget to enable these the bot will not work as expected.

## Usage

To start this bot from the commandline:

```bash
go run main.go <args>
```

After setting up the bot you can type the following command in any discord channel the bot is monitoring.

```
!enable_mention
```

Afterwards you will receive a direct message from the bot instructing you to change your name on teamspeak.

This is needed to create a mapping between your discord user and your teamspeak user.
Afterwards you can change your name back on teamspeak.

## Docker

A dockerfile has also been included to build and run this via docker
When running this bot via docker you should take in mind that a file `discordTeamspeakMapping.json` will be created in `/app`

### Args

```
discord_bot_token   # The token retrieved when creating a bot via the discord developer portal
discord_guild_id    # The id of your discord server you want to run this bot for
ts_server_id        # The server id of your teamspeak server (a teamspeak server can run multiple virtual servers)
ts_user             # The username of your ServerQuery account for teamspeak
ts_password         # The password of your ServerQuery account for teamspeak
ts_url              # The url of your teamspeak server including the query port, e.g.: 127.0.0.1:10011
ts_ignore_channel   # Any channel id you want to ignore when looking for online users. This argument can be used multiple times to ignore multiple channels.
```

### Environment variables

All of the arguments can also be used as environment variables. For that to work you need to capitalize the argument name. Example: DISCORD_BOT_TOKEN=XXXX.

## Discord permissions

To control which channels this bot should monitor you should change the permissions of the bot after adding it to your server.
To do this you can edit a channel and go to the `Permissions` tab. Under `Advanced permissions` you can click on the `+` icon to add specific permissions for a role, select the `teamspeak-notifier` role. Via that way you could exclude this channel from being monitored by the bot.

## How does this bot work?

This bot connects to your discord server via the discord api. For connecting to your teamspeak server it uses the telnet protocol, also called `ServerQuery` for teamspeak.

When initially launching this bot it fetches all online users from discord and teamspeak. It uses the mapping created when executing the `!enable_mention` command to determine which user is present on both.

After launching the bot it uses events to update any user presence. For example when your status on discord changes from online to away the bot detects this and will remove you from the "presence" list.
At that point you will not be mentioned anymore.
When your status changes back to online you will start receiving mentions again.

The same goes for teamspeak, when moving to any channel the bot is ignoring (via the ts_ignore_channel argument) you will be removed from the "presence" list.

Both discord and teamspeak will also monitor online / offline events and will add or remove users accordingly.
