package mews

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tim-online/go-mews/commands"
	"github.com/tim-online/go-mews/reservations"
	"github.com/tim-online/go-mews/spaces"
)

var (
	WebsocketURL = &url.URL{
		Scheme: "wss",
		Host:   "www.mews.li",
		Path:   "/ws/connector",
	}
	WebsocketURLDemo = &url.URL{
		Scheme: "wss",
		Host:   "demo.mews.li",
		Path:   "/ws/connector",
	}
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Time to wait before force close on connection.
	closeGracePeriod = 10 * time.Second
)

type Websocket struct {
	// HTTP client used to communicate with the DO API.
	client *http.Client

	// Base URL for API requests
	baseURL *url.URL

	// Debugging flag
	debug bool

	// Disallow unknown json fields
	disallowUnknownFields bool

	accessToken string
	clientToken string

	connection *websocket.Conn
	cancelFunc context.CancelFunc

	doneChan chan struct{}
	// msgChan  chan []byte
	errChan chan error

	cmdChan         chan CommandEvent
	resChan         chan ReservationEvent
	spaceChan       chan SpaceEvent
	priceUpdateChan chan PriceUpdateEvent
}

func NewWebsocket(httpClient *http.Client, accessToken string, clientToken string) *Websocket {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	ws := &Websocket{}
	ws.SetAccessToken(accessToken)
	ws.SetClientToken(clientToken)
	ws.SetDebug(false)
	ws.SetBaseURL(WebsocketURL)

	ws.doneChan = make(chan struct{})
	// ws.msgChan = make(chan []byte)
	ws.errChan = make(chan error)
	ws.cmdChan = make(chan CommandEvent)
	ws.resChan = make(chan ReservationEvent)
	ws.spaceChan = make(chan SpaceEvent)
	ws.priceUpdateChan = make(chan PriceUpdateEvent)

	return ws
}

func (ws Websocket) AccessToken() string {
	return ws.accessToken
}

func (ws *Websocket) SetAccessToken(accessToken string) {
	ws.accessToken = accessToken
}

func (ws Websocket) ClientToken() string {
	return ws.clientToken
}

func (ws *Websocket) SetClientToken(clientToken string) {
	ws.clientToken = clientToken
}

func (ws *Websocket) BaseURL() *url.URL {
	return ws.baseURL
}

func (ws *Websocket) SetBaseURL(baseURL *url.URL) {
	ws.baseURL = baseURL
	ws.baseURL.Scheme = "wss"
}

func (ws *Websocket) Debug() bool {
	return ws.debug
}

func (ws *Websocket) SetDebug(debug bool) {
	ws.debug = debug
}

func (ws *Websocket) CommandEvents() chan (CommandEvent) {
	return ws.cmdChan
}

func (ws *Websocket) ReservationEvents() chan (ReservationEvent) {
	return ws.resChan
}

func (ws *Websocket) SpaceEvents() chan (SpaceEvent) {
	return ws.spaceChan
}

func (ws *Websocket) PriceUpdateEvents() chan (PriceUpdateEvent) {
	return ws.priceUpdateChan
}

func (ws *Websocket) Errors() chan (error) {
	return ws.errChan
}

func (ws *Websocket) Connect(ctx context.Context) error {
	var err error

	u := ws.BaseURL()
	q := u.Query()
	q.Add("ClientToken", ws.ClientToken())
	q.Add("AccessToken", ws.AccessToken())
	u.RawQuery = q.Encode()

	ws.connection, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	// Time allowed to read the next pong message from the peer.
	ws.connection.SetReadDeadline(time.Now().Add(pongWait))
	// After receiving a pong: reset the read deadline
	ws.connection.SetPongHandler(func(string) error { ws.connection.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// Send ping messages. Stop doing that when context is canceled
	go func() {
		err := ws.KeepAlive(ctx)
		if err != nil {
			ws.errChan <- err
		}
	}()

	// Receive close messages from the peer
	ws.connection.SetCloseHandler(func(code int, text string) error {
		// ws.Close()
		return nil
	})

	// read messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			default:
				_, msg, err := ws.connection.ReadMessage()
				if err != nil {
					_, ok := err.(*websocket.CloseError)
					if !ok {
						ws.errChan <- err
					}
					return
				}

				message := Message{}
				err = json.Unmarshal(msg, &message)
				if err != nil {
					ws.errChan <- err
					return
				}

				for _, b := range message.Events {
					event := Event{}
					err = json.Unmarshal(b, &event)
					if err != nil {
						ws.errChan <- err
						return
					}

					if event.Type == EventTypeDeviceCommand && ws.cmdChan != nil {
						cmdEvent := CommandEvent{}
						err := json.Unmarshal(b, &cmdEvent)
						if err != nil {
							ws.errChan <- err
							return
						}
						ws.cmdChan <- cmdEvent
					}

					if event.Type == EventTypeReservation && ws.resChan != nil {
						resEvent := ReservationEvent{}
						err := json.Unmarshal(b, &resEvent)
						if err != nil {
							ws.errChan <- err
							return
						}
						ws.resChan <- resEvent
					} else if event.Type == EventTypeSpace && ws.spaceChan != nil {
						spaceEvent := SpaceEvent{}
						err := json.Unmarshal(b, &spaceEvent)
						if err != nil {
							ws.errChan <- err
							return
						}
						ws.spaceChan <- spaceEvent
					} else if event.Type == EventTypePriceUpdate && ws.priceUpdateChan != nil {
						priceUpdateEvent := PriceUpdateEvent{}
						err := json.Unmarshal(b, &priceUpdateEvent)
						if err != nil {
							ws.errChan <- err
							return
						}
						ws.priceUpdateChan <- priceUpdateEvent
					}
				}
			}
		}
	}()

	return nil
}

func (ws *Websocket) KeepAlive(ctx context.Context) error {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ws.Debug() {
				log.Println("sending keep alive ping message")
			}
			err := ws.connection.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
			if err != nil {
				return err
			}
		case <-ctx.Done():
			if ws.Debug() {
				log.Println("keep alive stopped")
			}
			return nil
		}
	}
}

func (ws *Websocket) Close() error {
	// Send close message to the peer
	if ws.Debug() {
		log.Println("send close message to peer")
	}
	message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	err := ws.connection.WriteControl(websocket.CloseMessage, message, time.Now().Add(writeWait))
	if err != nil {
		return err
	}

	// wait for a specified time before force-closing the connection
	time.Sleep(closeGracePeriod)

	// Close closes the underlying network connection without sending or waiting for a close message.
	return ws.connection.Close()
}

func (ws *Websocket) ReadMessages() {
}

func (ws *Websocket) Stop() {
	ws.doneChan <- struct{}{}
	ws.connection.Close()
}

type Message struct {
	Events []json.RawMessage `json:"Events"`
}

type Events []Event

type Event struct {
	Type  EventType             `json:"Type"`  // Type of the event.
	ID    string                `json:"Id"`    // Unique identifier of the Command.
	State commands.CommandState `json:"State"` // State of the command.
}

type EventType string

const (
	EventTypeDeviceCommand EventType = "DeviceCommand"
	EventTypeReservation   EventType = "Reservation"
	EventTypeSpace         EventType = "Space"
	EventTypePriceUpdate   EventType = "PriceUpdate"
)

type CommandEvent struct {
	Event
}

type ReservationEvent struct {
	Event

	ID              string                        `json:"Id"`              // Unique identifier of the Reservation.
	State           reservations.ReservationState `json:"State"`           // State of the reservation.
	StartUTC        time.Time                     `json:"StartUtc"`        // Start of the reservation (arrival) in UTC timezone in ISO 8601 format.
	EndUTC          time.Time                     `json:"EndUtc"`          // End of the reservation (departure) in UTC timezone in ISO 8601 format.
	AssignedSpaceID string                        `json:"AssignedSpaceId"` // Unique identifier of the operations/enterprises#space assigned to the reservation.
}

type SpaceEvent struct {
	Event

	State spaces.SpaceState `json:"State"` // State of the space.

}

type PriceUpdateEvent struct {
	Event

	StartUTC        time.Time `json:"StartUtc"`        // Start of the price update interval in UTC timezone in ISO 8601 format.
	EndUtc          time.Time `json:"EndUtc"`          // End of the price update interval in UTC timezone in ISO 8601 format.
	RateID          string    `json:"RateId"`          // Unique identifier of the Rate assigned to the update price event.
	SpaceCategoryID string    `json:"SpaceCategoryId"` // Unique identifier of the Space category assigned to the update price event.
}