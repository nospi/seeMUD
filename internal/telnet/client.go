package telnet

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Client represents a telnet connection to a MUD server
type Client struct {
	host       string
	port       string
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	connected  bool
	mutex      sync.RWMutex
	outputChan chan string
	inputChan  chan string
	closeChan  chan bool
}

// NewClient creates a new telnet client
func NewClient(host, port string) *Client {
	return &Client{
		host:       host,
		port:       port,
		outputChan: make(chan string, 100),
		inputChan:  make(chan string, 10),
		closeChan:  make(chan bool, 1),
	}
}

// Connect establishes connection to the MUD server
func (c *Client) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	address := net.JoinHostPort(c.host, c.port)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.writer = bufio.NewWriter(conn)
	c.connected = true

	// Start goroutines for reading and writing
	go c.readLoop()
	go c.writeLoop()

	return nil
}

// Disconnect closes the connection
func (c *Client) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	close(c.closeChan)

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected
}

// SendCommand sends a command to the MUD server
func (c *Client) SendCommand(command string) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	select {
	case c.inputChan <- command:
		return nil
	default:
		return fmt.Errorf("input buffer full")
	}
}

// GetOutput returns the output channel for reading server messages
func (c *Client) GetOutput() <-chan string {
	return c.outputChan
}

// readLoop continuously reads from the server
func (c *Client) readLoop() {
	defer func() {
		c.mutex.Lock()
		c.connected = false
		c.mutex.Unlock()
	}()

	for {
		select {
		case <-c.closeChan:
			return
		default:
			if c.conn != nil {
				c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				line, err := c.reader.ReadString('\n')
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}
					// Connection lost or other error
					return
				}

				// Clean up the line and send to output channel
				line = strings.TrimRight(line, "\r\n")
				if line != "" {
					select {
					case c.outputChan <- line:
					default:
						// Output buffer full, skip this line
					}
				}
			}
		}
	}
}

// writeLoop continuously writes to the server
func (c *Client) writeLoop() {
	for {
		select {
		case <-c.closeChan:
			return
		case command := <-c.inputChan:
			if c.conn != nil && c.connected {
				c.writer.WriteString(command + "\n")
				c.writer.Flush()
			}
		}
	}
}