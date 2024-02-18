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

type Client struct {
	address  string
	password string
	conn     *rcon.Conn
}

func NewClient(address string, password string) *Client {
	client := &Client{
		address:  address,
		password: password,
	}
	return client
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

func (r *Client) BanPlayer(steamID uint64) error {
	response, err := r.executeWithRetry(fmt.Sprintf("BanPlayer %d", steamID), true)
	if err != nil {
		return err
	} else if !strings.HasPrefix(response, "Baned: ") { // sic!
		return fmt.Errorf("failed to ban player: %s", response)
	}
	return nil
}

func (r *Client) Broadcast(message string) error {
	response, err := r.executeWithRetry(fmt.Sprintf("Broadcast %s", message), true)
	if err != nil {
		return err
	} else if !strings.HasPrefix(response, "Broadcasted: ") {
		return fmt.Errorf("failed to broadcast: %s", response)
	}
	return nil
}

func (r *Client) DoExit() error {
	response, err := r.executeWithRetry("DoExit", true)
	if err != nil {
		return err
	} else if response != "Shutdown..." {
		return fmt.Errorf("failed to shut down: %s", response)
	}
	return nil
}

type ServerInfo struct {
	ServerName string
	Version    string
}

var infoRegex = regexp.MustCompile(`^Welcome to Pal Server\[v([\d\.]+)\]\s*(.*?)$`)

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

func (r *Client) KickPlayer(steamID uint64) error {
	response, err := r.executeWithRetry(fmt.Sprintf("KickPlayer %d", steamID), true)
	if err != nil {
		return err
	} else if !strings.HasPrefix(response, "Kicked: ") {
		return fmt.Errorf("failed to kick player: %s", response)
	}
	return nil
}

func (r *Client) Save() error {
	response, err := r.executeWithRetry("Save", true)
	if err != nil {
		return err
	} else if response != "Complete Save" {
		return fmt.Errorf("failed to save: %s", response)
	}
	return nil
}

type Player struct {
	Name      string
	PlayerUID uint64
	SteamID   uint64
}

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

func (r *Client) Shutdown(seconds int) error {
	return r.shutdown(fmt.Sprintf("%d", seconds))
}

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
