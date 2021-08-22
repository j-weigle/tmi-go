package tmi

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type clientConfig struct {
	Connection connectionConfig // how the client will connect and reconnect.
	Identity   identityConfig   // who the client logs in as.
	Pinger     pingConfig       // how often to ping, and timeout.
}

type connectionConfig struct {
	reconnect            bool          // if true, reconnect on reconnect requests and non-fatal errors.
	secure               bool          // if true, connect to to Twitch's secure server(port 443), otherwise insecure (port 80).
	maxReconnectAttempts int           // maximum number of attempts to reconnect when disconnected.
	maxReconnectInterval time.Duration // maximum interval between reconnect attempts.
}

type identityConfig struct {
	username string // login account name
	password string // oauth token
}

type pingConfig struct {
	enabled  bool          // whether to send pings or not
	interval time.Duration // how long to wait before sending a ping when no messages have been received
	timeout  time.Duration // how long to wait on a pong before reconnecting
}

// NewClientConfig returns a client config with Connection settings initialzed
// to the recommended defaults. Identity is initialzed but left empty.
func NewClientConfig() clientConfig {
	conn := connectionConfig{}
	conn.Default()
	pinger := pingConfig{}
	pinger.Default()
	return clientConfig{
		Connection: conn,
		Identity:   identityConfig{},
		Pinger:     pinger,
	}
}

// Default sets the connection configuration options to their recommended defaults.
//
// Default options:
// reconnect            = true,
// secure               = true,
// maxReconnectAttempts = -1 (infinite),
// maxReconnectInterval = 30 seconds,
func (c *connectionConfig) Default() {
	c.reconnect = true
	c.secure = true
	c.maxReconnectAttempts = -1
	c.maxReconnectInterval = time.Second * 30
}

// SetReconnect sets whether the client will attempt to reconnect
// to the server in the case of a disconnect.
func (c *connectionConfig) SetReconnect(reconnect bool) {
	c.reconnect = reconnect
}

// SetReconnectSettings sets how often and how many times the client
// will attempt to reconnect to the server in the case of a disconnect.
func (c *connectionConfig) SetReconnectSettings(maxAttempts int, maxInterval time.Duration) {
	c.maxReconnectAttempts = maxAttempts
	if maxInterval < time.Second*5 {
		maxInterval = time.Second * 5
	}
	c.maxReconnectInterval = maxInterval
}

// SetSecure sets the connection scheme and port.
// true uses scheme = wss and port = 443.
// false uses scheme = ws and port = 80.
func (c *connectionConfig) SetSecure(secure bool) {
	c.secure = secure
}

// Anonymous sets username to an random justinfan username (password can be anything).
func (id *identityConfig) Anonymous() {
	id.username = "justinfan" + fmt.Sprint(rand.Intn(79000)+1000)
	id.password = "swordfish"
}

// Set sets the login identity configuration to username and password with oauth: prepended.
func (id *identityConfig) Set(username, password string) {
	id.SetUsername(username)
	id.SetPassword(password)
}

// SetPassword sets the password for the identity configuration to password with oauth: prepended.
func (id *identityConfig) SetPassword(password string) {
	password = strings.TrimSpace(password)
	if !strings.HasPrefix(password, "oauth:") {
		password = "oauth:" + password
	}
	id.password = password
}

// SetUsername sets the username for the identity configuration to username.
func (id *identityConfig) SetUsername(username string) {
	username = strings.TrimSpace(username)
	id.username = strings.ToLower(username)
}

// Default sets the ping configuration options to their recommended defaults.
func (p *pingConfig) Default() {
	p.enabled = true
	p.interval = time.Minute
	p.timeout = time.Second * 5
}

// Disable sending pings
func (p *pingConfig) Disable() {
	p.enabled = false
}

// Enable sending pings
func (p *pingConfig) Enable() {
	p.enabled = true
}

// Set sets the idle wait time and timeout for ping configuration.
func (p *pingConfig) Set(interval, timeout time.Duration) {
	p.interval = interval
	p.timeout = timeout
}
