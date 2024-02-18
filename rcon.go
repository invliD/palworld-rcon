package palworldrcon

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/gorcon/rcon"
)

// Client represents a client to a Palword RCON server. Only use [NewClient] to instantiate a client.
//
// Available commands are documented at https://tech.palworldgame.com/settings-and-operation/commands#command-list.
// Not all commands listed there are applicable from RCON.
type Client struct {
	address  string
	password string
	conn     *rcon.Conn
}

// NewClient creates a new [Client] with the given address and password. Note that this function does not attempt to
// connect to the RCON server, so the password is not validated at this time. Instead, the connection is established
// on-demand when a method on the [Client] is called.
func NewClient(address string, password string) *Client {
	client := &Client{
		address:  address,
		password: password,
	}
	return client
}

// Close closes the connection to the RCON server. Calling any other method after closing will reopen the connection.
func (r *Client) Close() error {
	if r.conn == nil {
		return nil
	} else {
		err := r.conn.Close()
		r.conn = nil
		return err
	}
}

func (r *Client) connect() error {
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	conn, err := rcon.Dial(r.address, r.password)
	if err != nil {
		return err
	}
	r.conn = conn
	return nil
}

func (r *Client) executeWithRetry(command string, retry bool) (string, error) {
	if r.conn == nil {
		err := r.connect()
		if err != nil {
			return "", err
		}
	}
	response, err := r.conn.Execute(command)
	if (errors.Is(err, io.EOF) || errors.Is(err, syscall.EPIPE)) && retry {
		err := r.connect()
		if err != nil {
			return "", err
		}
		return r.executeWithRetry(command, false)
	}
	return strings.TrimSpace(response), err
}

// KickPlayer instructs the server to ban the player with the given Steam ID from the server. The player must be online
// to be banned.
func (r *Client) BanPlayer(steamID uint64) error {
	response, err := r.executeWithRetry(fmt.Sprintf("BanPlayer %d", steamID), true)
	if err != nil {
		return err
	} else if !strings.HasPrefix(response, "Baned: ") { // sic!
		return fmt.Errorf("failed to ban player: %s", response)
	}
	return nil
}

// Broadcast displays the given message to all online players.
func (r *Client) Broadcast(message string) error {
	response, err := r.executeWithRetry(fmt.Sprintf("Broadcast %s", message), true)
	if err != nil {
		return err
	} else if !strings.HasPrefix(response, "Broadcasted: ") {
		return fmt.Errorf("failed to broadcast: %s", response)
	}
	return nil
}

// DoExit instructs the server to immediately exit.
func (r *Client) DoExit() error {
	response, err := r.executeWithRetry("DoExit", true)
	if err != nil {
		return err
	} else if response != "Shutdown..." {
		return fmt.Errorf("failed to shut down: %s", response)
	}
	return nil
}

// ServerInfo represents the information about the server returned by [Client.Info]().
type ServerInfo struct {
	ServerName string
	Version    string
}

var infoRegex = regexp.MustCompile(`^Welcome to Pal Server\[v([\d\.]+)\]\s*(.*?)$`)

// Info returns information about the server.
func (r *Client) Info() (*ServerInfo, error) {
	response, err := r.executeWithRetry("Info", true)
	if err != nil {
		return nil, err
	}
	infoMatches := infoRegex.FindStringSubmatch(response)
	if infoMatches == nil {
		return nil, fmt.Errorf("failed to parse Info output: %s", response)
	}
	return &ServerInfo{
		ServerName: infoMatches[2],
		Version:    infoMatches[1],
	}, nil
}

// KickPlayer instructs the server to kick the player with the given Steam ID.
func (r *Client) KickPlayer(steamID uint64) error {
	response, err := r.executeWithRetry(fmt.Sprintf("KickPlayer %d", steamID), true)
	if err != nil {
		return err
	} else if !strings.HasPrefix(response, "Kicked: ") {
		return fmt.Errorf("failed to kick player: %s", response)
	}
	return nil
}

// Save instructs the server to save the world to disk.
func (r *Client) Save() error {
	response, err := r.executeWithRetry("Save", true)
	if err != nil {
		return err
	} else if response != "Complete Save" {
		return fmt.Errorf("failed to save: %s", response)
	}
	return nil
}

// Player is the representation of a single player.
type Player struct {
	Name      string
	PlayerUID uint64
	SteamID   uint64
}

// ShowPlayers returns a list of all players that are currently online.
func (r *Client) ShowPlayers() ([]Player, error) {
	players := []Player{}
	response, err := r.executeWithRetry("ShowPlayers", true)
	if err != nil {
		return players, err
	}
	c := csv.NewReader(strings.NewReader(strings.TrimSpace(response)))
	records, err := c.ReadAll()
	if err != nil {
		return players, fmt.Errorf("failed to parse ShowPlayers response as CSV: %w", err)
	}
	for _, record := range records[1:] {
		if len(record) != 3 {
			return players, fmt.Errorf("failed to parse player output: %v", record)
		}
		playerUID, err := strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			return players, fmt.Errorf("failed to parse player UID: %w", err)
		}
		steamID, err := strconv.ParseUint(record[2], 10, 64)
		if err != nil {
			return players, fmt.Errorf("failed to parse steam ID: %w", err)
		}
		players = append(players, Player{
			Name:      record[0],
			PlayerUID: playerUID,
			SteamID:   steamID,
		})
	}
	return players, nil
}

// Shutdown instructs the server to shut down after the given number of seconds.
func (r *Client) Shutdown(seconds int) error {
	return r.shutdown(fmt.Sprintf("%d", seconds))
}

// ShutdownWithMessage instructs the server to shut down after the given number of seconds. The message will be
// displayed to all online players.
func (r *Client) ShutdownWithMessage(seconds int, message string) error {
	return r.shutdown(fmt.Sprintf("%d %s", seconds, message))
}

func (r *Client) shutdown(args string) error {
	response, err := r.executeWithRetry(fmt.Sprintf("Shutdown %s", args), true)
	if err != nil {
		return nil
	} else if !strings.HasPrefix(response, "The server will shut down") {
		return fmt.Errorf("failed to shut down: %s", response)
	}
	return nil
}
