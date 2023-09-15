package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type simpleCounter struct {
	work int
}

func (s simpleCounter) GetCounter() int {
	time.Sleep(time.Second * time.Duration(s.work))
	return 12
}

func (s simpleCounter) SaveCounter(path string) error {
	return nil
}

func TestRunHttpServer(t *testing.T) {
	srv := runHttpServer(simpleCounter{1}, 5)
	t.Cleanup(func() {
		srv.Shutdown(context.TODO())
		// http.DefaultServeMux seems to outlive tests, needs to be wiped for next test
		// initilized as in http/server.go:2338
		http.DefaultServeMux = new(http.ServeMux)
	})
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

// Tests whether only 2 threads are running in parallel
func TestRunHttpServerWithParallelLimit(t *testing.T) {
	srv := runHttpServer(simpleCounter{3}, 2)
	t.Cleanup(func() {
		srv.Shutdown(context.TODO())
		// http.DefaultServeMux seems to outlive tests, needs to be wiped for next test
		// initilized as in http/server.go:2338
		http.DefaultServeMux = new(http.ServeMux)
	})
	//give the server a second
	time.Sleep(time.Second * 1)

	start := time.Now()
	responseTimes := make(chan time.Time, 5)
	for i := 0; i < 5; i++ {
		go func() {
			resp, err := http.Get("http://127.0.0.1:8080")
			if err != nil {
				t.Errorf("could not call GET on running http server: %s", err.Error())
			}
			responseGotten := time.Now()
			responseTimes <- responseGotten
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("could not read response from http server: %s", err.Error())
			}

			if strings.TrimSpace(string(body)) != strings.TrimSpace(fmt.Sprintf("12 requests in the last %d seconds", windowSizeInSeconds)) {
				t.Errorf("response was not correct: %s", string(body))
			}
		}()
	}

	immediate := 2
	afterThree := 2
	afterSix := 1

	// 5 times since we expect 5 responses
	for i := 0; i < 5; i++ {
		respTime := <-responseTimes
		// - 3.0 -> substract "work" (sleep) time
		secondsSinceStart := respTime.Sub(start).Seconds() - 3.0
		switch {
		case secondsSinceStart < 1.0:
			immediate--
		case secondsSinceStart >= 3.0 && secondsSinceStart < 4.0:
			afterThree--
		case secondsSinceStart >= 6.0 && secondsSinceStart < 7.0:
			afterSix--
		}
	}

	if immediate+afterThree+afterSix != 0 {
		t.Errorf("request did not run parallel as expected:\nimmediate: %d\nafter three seconds: %d\nafter six seconds: %d\n",
			immediate, afterThree, afterSix)
	}
}
