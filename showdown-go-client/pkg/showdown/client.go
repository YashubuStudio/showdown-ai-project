package showdown

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Line struct {
	RoomID string
	Raw    string
}

type QueryResponse struct {
	Type string
	Raw  json.RawMessage
}

type Client struct {
	baseURL  *url.URL
	username string

	conn    *websocket.Conn
	writeMu sync.Mutex // serializes WebSocket writes

	mu           sync.RWMutex
	lines        chan Line
	errCh        chan error
	doneCh       chan struct{}
	connected    bool
	named        bool
	currentUser  string
	formats      []string
	queryWaiters map[string]chan QueryResponse
	readErr      error
	closeOnce    sync.Once
	doneOnce     sync.Once
}

func NewClient(baseURL, username string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "http"
	}
	if parsed.Host == "" {
		parsed.Host = parsed.Path
		parsed.Path = ""
	}
	return &Client{
		baseURL:      parsed,
		username:     username,
		lines:        make(chan Line, 512),
		errCh:        make(chan error, 16),
		doneCh:       make(chan struct{}),
		queryWaiters: make(map[string]chan QueryResponse),
	}, nil
}

func (c *Client) Connect(ctx context.Context) error {
	wsURL := *c.baseURL
	switch wsURL.Scheme {
	case "https":
		wsURL.Scheme = "wss"
	default:
		wsURL.Scheme = "ws"
	}
	wsURL.Path = "/showdown/websocket"

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL.String(), nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	go c.readLoop()
	return nil
}

func (c *Client) readLoop() {
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.connected = false
			c.readErr = err
			c.mu.Unlock()
			c.pushErr(err)
			c.closeDone()
			return
		}
		c.handleFrame(string(payload))
	}
}

func (c *Client) handleFrame(frame string) {
	frame = strings.TrimSpace(frame)
	if frame == "" {
		return
	}

	roomID := ""
	lines := strings.Split(frame, "\n")
	if strings.HasPrefix(lines[0], ">") {
		roomID = strings.TrimPrefix(lines[0], ">")
		lines = lines[1:]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		c.handleLine(roomID, line)
	}
}

func (c *Client) handleLine(roomID, line string) {
	if strings.HasPrefix(line, "|updateuser|") {
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			c.mu.Lock()
			c.currentUser = strings.TrimSpace(parts[2])
			c.named = parts[3] == "1"
			c.mu.Unlock()
		}
	}

	if strings.HasPrefix(line, "|formats") {
		out := parseFormatsLine(line)
		if len(out) > 0 {
			c.mu.Lock()
			c.formats = out
			c.mu.Unlock()
		}
	}

	if strings.HasPrefix(line, "|queryresponse|") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			qr := QueryResponse{Type: parts[2], Raw: json.RawMessage(parts[3])}
			c.mu.RLock()
			waiter := c.queryWaiters[qr.Type]
			c.mu.RUnlock()
			if waiter != nil {
				select {
				case waiter <- qr:
				default:
				}
			}
		}
	}

	select {
	case c.lines <- Line{RoomID: roomID, Raw: line}:
	default:
		log.Printf("showdown: line channel full, dropping: %s", line)
	}
}

func (c *Client) pushErr(err error) {
	select {
	case c.errCh <- err:
	default:
	}
}

func (c *Client) Errors() <-chan error {
	return c.errCh
}

func (c *Client) Lines() <-chan Line {
	return c.lines
}

func (c *Client) Done() <-chan struct{} {
	return c.doneCh
}

func (c *Client) Rename(ctx context.Context) error {
	return c.Send(ctx, "", fmt.Sprintf("/trn %s", c.username))
}

func (c *Client) Send(ctx context.Context, roomID, text string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return errors.New("not connected")
	}

	payload := text
	if roomID != "" {
		payload = roomID + "|" + text
	} else {
		payload = "|" + text
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	deadline, ok := ctx.Deadline()
	if ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}
	return conn.WriteMessage(websocket.TextMessage, []byte(payload))
}

func (c *Client) Query(ctx context.Context, query string) (json.RawMessage, error) {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return nil, errors.New("query is empty")
	}
	queryType := fields[0]

	waiter := make(chan QueryResponse, 1)
	c.mu.Lock()
	if _, exists := c.queryWaiters[queryType]; exists {
		c.mu.Unlock()
		return nil, fmt.Errorf("query %q is already in flight", queryType)
	}
	c.queryWaiters[queryType] = waiter
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.queryWaiters, queryType)
		c.mu.Unlock()
	}()

	if err := c.Send(ctx, "", "/query "+query); err != nil {
		return nil, err
	}

	select {
	case resp := <-waiter:
		return resp.Raw, nil
	case <-c.doneCh:
		if err := c.connectionErr(); err != nil {
			return nil, err
		}
		return nil, errors.New("connection closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.mu.Lock()
		conn := c.conn
		c.conn = nil
		c.connected = false
		c.mu.Unlock()
		if conn != nil {
			err = conn.Close()
		}
		c.closeDone()
	})
	return err
}

func (c *Client) Username() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.currentUser != "" {
		return c.currentUser
	}
	return c.username
}

func (c *Client) Named() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.named
}

func (c *Client) Formats() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, len(c.formats))
	copy(out, c.formats)
	return out
}

func (c *Client) connectionErr() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.readErr
}

func (c *Client) closeDone() {
	c.doneOnce.Do(func() {
		close(c.doneCh)
	})
}

var usernameCounter atomic.Uint64

func randomUsername(prefix string) string {
	return fmt.Sprintf("%s-%d-%06d", prefix, time.Now().UnixNano(), usernameCounter.Add(1)%1000000)
}

func parseFormatsLine(line string) []string {
	parts := strings.Split(line, "|")
	if len(parts) < 3 {
		return nil
	}

	out := make([]string, 0, len(parts)-2)
	skipSectionName := false
	for _, format := range parts[2:] {
		format = strings.TrimSpace(format)
		if format == "" {
			continue
		}
		if strings.HasPrefix(format, ",") {
			skipSectionName = true
			continue
		}
		if skipSectionName {
			skipSectionName = false
			continue
		}
		name := strings.Split(format, ",")[0]
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}
