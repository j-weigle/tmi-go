package tmi

import (
	"errors"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	twitchWSSHost = "irc-ws.chat.twitch.tv:443"
	twitchWSHost  = "irc-ws.chat.twitch.tv:80"
)

var (
	errReconnect        = errors.New("reconnect")
	ErrDisconnectCalled = errors.New("disconnect was called")
	ErrLoginFailure     = errors.New("login failure")
)

// Connect connects to irc-ws.chat.twitch.tv and attempts to reconnect on connection errors.
func (c *Client) Connect() error {
	var err error
	var u url.URL

	if c.config.Connection.secure {
		u = url.URL{Scheme: "wss", Host: twitchWSSHost}
	} else {
		u = url.URL{Scheme: "ws", Host: twitchWSHost}
	}

	var maxReconnectAttempts int = c.config.Connection.maxReconnectAttempts
	var maxReconnectInterval time.Duration = c.config.Connection.maxReconnectInterval

	// Reset disconnect before starting connection loop. connect() will check if it has
	// been used before attempting to (re)connect.
	c.notifDisconnect.reset()

	for {
		err = c.connect(u)

		switch err {
		case errReconnect:
			const overflowPoint = 64 // technically 63, but using i - 1

			var i int = c.reconnectCounter
			c.reconnectCounter++
			// in case of overflow, reset to overflow point in order to maintain max interval
			if c.reconnectCounter < 0 {
				c.reconnectCounter = overflowPoint
			}

			if maxReconnectAttempts >= 0 && i >= maxReconnectAttempts {
				err = errors.New("max attempts to reconnect reached")
				c.callDone(err)
				return err
			}

			var sleepDuration time.Duration
			if i == 0 {
				continue // immediate reconnect on first attempt
			} else if i > 0 && i < overflowPoint {
				// i - 1 because math.Pow(2, 0) == 1
				sleepDuration = time.Duration(math.Pow(2, float64(i-1)))
			} else {
				sleepDuration = maxReconnectInterval
			}

			if sleepDuration > maxReconnectInterval {
				sleepDuration = maxReconnectInterval
			}

			time.Sleep(sleepDuration)

		default:
			c.callDone(err)
			return err
		}
	}
}

// Disconnect closes the connection to the server, and does not attempt to reconnect.
func (c *Client) Disconnect() {
	c.notifDisconnect.notify()
}

// Join joins channel.
func (c *Client) Join(channels ...string) error {
	if channels == nil || len(channels) < 1 {
		return errors.New("channels was empty or nil")
	}

	var newJoins = []string{}
	c.channelsMutex.Lock()
	for _, channel := range channels {
		channel = formatChannel(channel)

		connected, ok := c.channels[channel]
		if !ok {
			c.channels[channel] = false
		}
		if !connected {
			newJoins = append(newJoins, channel)
		}
	}
	c.channelsMutex.Unlock()

	if c.connected.get() {
		if len(newJoins) > 0 {
			go c.joinChannels(newJoins)
		}
	}
	return nil
}

// Done sets the callback function for when a client is done to cb. Useful for running a client in a goroutine.
func (c *Client) OnDone(cb func(fatal error)) {
	c.done = cb
}

// OnErr sets the callback function for general error messages to cb.
func (c *Client) OnErr(cb func(error)) {
	c.onError = cb
}

// Part leaves channels.
func (c *Client) Part(channels ...string) error {
	if channels == nil || len(channels) < 1 {
		return errors.New("channels was empty or nil")
	}

	for _, channel := range channels {
		channel = formatChannel(channel)

		if c.connected.get() {
			c.send("PART " + channel)
		}

		c.channelsMutex.Lock()
		delete(c.channels, channel)
		c.channelsMutex.Unlock()
	}

	return nil
}

// Say sends a PRIVMSG message in channel.
func (c *Client) Say(channel string, message string) error {
	channel = formatChannel(channel)

	if len(message) >= 500 {
		var messages = splitChatMessage(message)
		for _, m := range messages {
			c.send("PRIVMSG " + channel + " :" + m)
		}
	} else {
		c.send("PRIVMSG " + channel + " :" + message)
	}
	return nil
}

// UpdatePassword updates the password the client uses for authentication.
func (c *Client) UpdatePassword(password string) {
	c.config.Identity.SetPassword(password)
}

func (c *Client) OnUnsetMessage(cb func(UnsetMessage)) {
	c.handlers.onUnsetMessage = cb
}
func (c *Client) OnConnected(cb func()) {
	c.handlers.onConnected = cb
}
func (c *Client) OnClearChatMessage(cb func(ClearChatMessage)) {
	c.handlers.onClearChatMessage = cb
}
func (c *Client) OnClearMsgMessage(cb func(ClearMsgMessage)) {
	c.handlers.onClearMsgMessage = cb
}
func (c *Client) OnGlobalUserstateMessage(cb func(GlobalUserstateMessage)) {
	c.handlers.onGlobalUserstateMessage = cb
}
func (c *Client) OnHostTargetMessage(cb func(HostTargetMessage)) {
	c.handlers.onHostTargetMessage = cb
}
func (c *Client) OnNoticeMessage(cb func(NoticeMessage)) {
	c.handlers.onNoticeMessage = cb
}
func (c *Client) OnReconnectMessage(cb func(ReconnectMessage)) {
	c.handlers.onReconnectMessage = cb
}
func (c *Client) OnRoomstateMessage(cb func(RoomstateMessage)) {
	c.handlers.onRoomstateMessage = cb
}
func (c *Client) OnUserNoticeMessage(cb func(UsernoticeMessage)) {
	c.handlers.onUserNoticeMessage = cb
}
func (c *Client) OnUserstateMessage(cb func(UserstateMessage)) {
	c.handlers.onUserstateMessage = cb
}
func (c *Client) OnNamesMessage(cb func(NamesMessage)) {
	c.handlers.onNamesMessage = cb
}
func (c *Client) OnJoinMessage(cb func(JoinMessage)) {
	c.handlers.onJoinMessage = cb
}
func (c *Client) OnPartMessage(cb func(PartMessage)) {
	c.handlers.onPartMessage = cb
}
func (c *Client) OnPingMessage(cb func(PingMessage)) {
	c.handlers.onPingMessage = cb
}
func (c *Client) OnPongMessage(cb func(PongMessage)) {
	c.handlers.onPongMessage = cb
}
func (c *Client) OnPrivateMessage(cb func(PrivateMessage)) {
	c.handlers.onPrivateMessage = cb
}
func (c *Client) OnWhisperMessage(cb func(WhisperMessage)) {
	c.handlers.onWhisperMessage = cb
}

func formatChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	if !strings.HasPrefix(channel, "#") {
		channel = "#" + channel
	}
	return strings.ToLower(channel)
}

func splitChatMessage(message string) []string {
	const splIdx = 500
	var messages []string

	for len(message) >= splIdx {
		var lastSpace = strings.LastIndex(message[:splIdx], " ")
		if lastSpace == -1 {
			lastSpace = splIdx
		}
		messages = append(messages, strings.TrimSpace(message[:lastSpace]))
		message = strings.TrimSpace(message[lastSpace:])
	}
	if message != "" {
		messages = append(messages, message)
	}

	return messages
}
