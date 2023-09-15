package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"simplesurance/requestwindow"
)

// configuration on production
const requestWindowFilePath = "./currentWindow"
const windowSizeInSeconds = 60
const requestSleepSeconds = 2
const allowedParallelThreads = 5

type Counter interface {
	GetCounter() int
	SaveCounter(path string) error
}

func runHttpServer(counter Counter, allowedParallels int) *http.Server {
	// semaphore is a channel that will allow up to n operations at once.
	var semaphore = make(chan int, allowedParallels)
	h := func(w http.ResponseWriter, _ *http.Request) {
		semaphore <- 1
		count := counter.GetCounter()
		<-semaphore
		io.WriteString(w, fmt.Sprintf("%d requests in the last %d seconds\n", count, windowSizeInSeconds))
	}
	http.HandleFunc("/", h)

	// however the server stops, save the current window
	defer func() {
		err := counter.SaveCounter(requestWindowFilePath)
		if err != nil {
			log.Printf("unable to save current window: %s", err.Error())
		}
	}()
	srv := &http.Server{Addr: ":8080"}
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Printf("server stoped operating: %s", err.Error())
		}
	}()

	return srv
}

func main() {
	var rw *requestwindow.RequestWindow
	if _, err := os.Stat(requestWindowFilePath); errors.Is(err, os.ErrNotExist) {
		rw = requestwindow.NewWindow(nil, windowSizeInSeconds, requestSleepSeconds)
	} else {
		rw, err = requestwindow.NewWindowFromFile(requestWindowFilePath, windowSizeInSeconds, requestSleepSeconds)
		if err != nil {
			log.Fatalf("unable to create request window from file: %s\nconsider delting file", err.Error())
		}
	}

	srv := runHttpServer(rw, allowedParallelThreads)

	sigChannel := make(chan os.Signal)
	signal.Notify(sigChannel, os.Interrupt)
	// save request window on SIGINT
	<-sigChannel
	srv.Shutdown(context.TODO())
	rw.SaveCounter(requestWindowFilePath)
	os.Exit(0)
}
