package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/dertiedemann/scn-pick-ban/pkg/lcu"
	log "github.com/sirupsen/logrus"
)

func main() {

	log.SetLevel(log.DebugLevel)

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

	lcuOptions := lcu.Options{Token: token, Protocol: protocol, Port: uint16(port)}
	_, err = lcu.NewHttpClient(lcuOptions)
	if err != nil {
		log.Fatal(err)
	}

	wsClient, err := lcu.NewWebsocketClient(lcuOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer wsClient.Close()

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancelFunc := context.WithCancel(context.Background())

	messages, err := wsClient.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-interrupt:
			log.Info("Shutting down")
			cancelFunc()
			return
		case m := <-messages:
			log.Info(m)
		}
	}

}
