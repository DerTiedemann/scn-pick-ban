package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/dertiedemann/scn-pick-ban/pkg/lcu"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func main() {
	lockfilePath := "/home/dertiedemann/Games/league-of-legends/drive_c/Riot Games/League of Legends/lockfile"
	for _, err := os.Stat(lockfilePath); err != nil && err == os.ErrNotExist; {
		log.Info("Waiting for league client!")
		time.Sleep(2 * time.Second)
	}
	lockfileBytes, err := ioutil.ReadFile(lockfilePath)
	if err != nil {
		log.Fatal(err)
	}
	lockfileContents := strings.Split(string(lockfileBytes), ":")
	token, protocol := lockfileContents[3], lockfileContents[4]

	port, _ := strconv.Atoi(lockfileContents[2])

	_, err = lcu.New(protocol, token, port)
	if err != nil {
		log.Fatal(err)
	}

	u := url.URL{Scheme: "wss", Host: fmt.Sprintf("127.0.0.1:%d", port)}
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	//dialer.HandshakeTimeout = time.Second

	b64Token := base64.StdEncoding.EncodeToString([]byte("riot:" + token))

	header := http.Header{
		"Authorization": {fmt.Sprintf("Basic %s", b64Token)},
	}
	log.Info(u.String())

	c, resp, err := dialer.Dial(u.String(), header)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	log.Info(resp)
	c.WriteJSON([]interface{}{5, "OnJsonApiEvent"})

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			if strings.Contains(string(message), "/lol-champ-select/v1/") {
				log.Printf(string(message))
			}
		}
	}()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
