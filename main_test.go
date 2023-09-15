package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type simpleCounter struct{}

func (s simpleCounter) GetCounter() int {
	return 12
}

func (s simpleCounter) SaveCounter(path string) error {
	return nil
}

func TestRunHttpServer(t *testing.T) {
	go runHttpServer(simpleCounter{})

	//give the server a second
	time.Sleep(time.Second * 1)

	resp, err := http.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Errorf("could not call GET on running http server: %s", err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("could not read response from http server: %s", err.Error())
	}

	if strings.TrimSpace(string(body)) != strings.TrimSpace(fmt.Sprintf("12 requests in the last %d seconds", windowSizeInSeconds)) {
		t.Errorf("response was not correct: %s", string(body))
	}
}
