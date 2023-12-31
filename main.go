package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"simplesurance/requestwindow"
)

const requestWindowFilePath = "./currentWindow"
const windowSizeInSeconds = 60

type Counter interface {
	GetCounter() int
	SaveCounter(path string) error
}

func runHttpServer(counter Counter) {
	h := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, fmt.Sprintf("%d requests in the last %d seconds\n", counter.GetCounter(), windowSizeInSeconds))
	}
	http.HandleFunc("/", h)

	// however the server stops, save the current window
	defer func() {
		err := counter.SaveCounter(requestWindowFilePath)
		if err != nil {
			log.Printf("unable to save current window: %s", err.Error())
		}
	}()
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Printf("server stoped operating: %s", err.Error())
	}

	return
}

func main() {
	var rw *requestwindow.RequestWindow
	if _, err := os.Stat(requestWindowFilePath); errors.Is(err, os.ErrNotExist) {
		rw = requestwindow.NewWindow(nil, windowSizeInSeconds)
	} else {
		rw, err = requestwindow.NewWindowFromFile(requestWindowFilePath, windowSizeInSeconds)
		if err != nil {
			log.Fatalf("unable to create request window from file: %s\nconsider delting file", err.Error())
		}
	}

	sigChannel := make(chan os.Signal)
	signal.Notify(sigChannel, os.Interrupt)
	// save request window on SIGINT
	go func() {
		<-sigChannel
		rw.SaveCounter(requestWindowFilePath)
		os.Exit(1)
	}()

	runHttpServer(rw)

}
