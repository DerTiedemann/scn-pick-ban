package lcu

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type WebsocketClient struct {
	token       string
	conn        *websocket.Conn
	baseUrl     url.URL
	messageChan chan *Message
	done        chan struct{}
	once        sync.Once
}

func (w *WebsocketClient) Close() error {
	w.once.Do(func() {
		close(w.messageChan)
	})
	defer w.conn.Close()
	err := w.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	select {
	case <-w.done:
	case <-time.After(time.Second):
	}
	return nil

}

func (w *WebsocketClient) Connect(ctx context.Context) (<-chan *Message, error) {
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	b64Token := base64.StdEncoding.EncodeToString([]byte("riot:" + w.token))

	header := http.Header{
		"Authorization": {fmt.Sprintf("Basic %s", b64Token)},
	}

	conn, resp, err := dialer.Dial(w.baseUrl.String(), header)
	if err != nil {
		return nil, errors.Wrap(err, "failed to establish connection")
	}

	if resp.StatusCode != 101 {
		return nil, errors.New("failed to upgrade websocket connection")
	}

	w.conn = conn

	// Subscribe to all events
	err = conn.WriteJSON([]interface{}{5, "OnJsonApiEvent"})
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe to events")
	}

	go func() {
		defer w.Close()
		for {
			select {
			case <-ctx.Done():
			case <-w.done:
				return

			default:
				m, err := w.poll()
				if err != nil {
					log.Info(err)
					continue
				}
				w.messageChan <- m
			}
		}
	}()

	return w.messageChan, nil
}

func (w *WebsocketClient) poll() (*Message, error) {

	msgType, rawJson, err := w.conn.ReadMessage()
	if err != nil {
		close(w.done)
		return nil, errors.Wrap(err, "failed to read message")
	}
	if len(rawJson) == 0 {
		return nil, errors.Errorf("got empty message with code %d, skipping", msgType)
	}

	var parsedJsonArray []interface{}

	d := json.NewDecoder(bytes.NewReader(rawJson))
	d.UseNumber()
	err = d.Decode(&parsedJsonArray)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse json message")
	}

	parsedJsonMap, ok := parsedJsonArray[2].(map[string]interface{})
	if !ok {
		return nil, errors.New("could not typecast given message")
	}
	var message Message
	err = mapstructure.Decode(parsedJsonMap, &message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create message struct")
	}

	return &message, nil
}

func NewWebsocketClient(options Options) (*WebsocketClient, error) {
	if err := options.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid options")
	}

	client := &WebsocketClient{
		baseUrl: url.URL{
			Scheme: "wss",
			Host:   fmt.Sprintf("127.0.0.1:%d", options.Port),
		},
		token:       options.Token,
		messageChan: make(chan *Message),
		done:        make(chan struct{}),
	}

	return client, nil
}
