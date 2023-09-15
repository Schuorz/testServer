package requestwindow

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

const requestWindowTimeFormat = "2006-01-02 15:04:05"

type RequestWindow struct {
	lock                sync.Mutex
	window              []time.Time
	wSizeSeconds        int
	requestSleepSeconds int
}

// NewWindowFromFile creates a new RequestWindow populating it with a window read from a file under path
func NewWindowFromFile(path string, windowSizeSeconds int, reqSleepSeconds int) (*RequestWindow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var startWindow []time.Time
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t, err := time.ParseInLocation(requestWindowTimeFormat, scanner.Text(), time.Local)
		if err != nil {
			return nil, err
		}
		startWindow = append(startWindow, t)
	}

	return NewWindow(startWindow, windowSizeSeconds, reqSleepSeconds), nil
}

// NewWindow creates a new RequestWindow optionally from a starting window if startWindow is not nil
func NewWindow(startWindow []time.Time, windowSizeSeconds int, reqSleepSeconds int) *RequestWindow {
	return &RequestWindow{
		window:              startWindow,
		wSizeSeconds:        windowSizeSeconds,
		requestSleepSeconds: reqSleepSeconds,
	}
}

// GetCounter removes all entries in the window that were created prior 60 secondes ago, adds a new one
// for the caller and returns the count of entries in the window
func (r *RequestWindow) GetCounter() int {

	// now should only be set after the lock is acquired, this should keep the times in order
	now := time.Now()

	// sleep for as long as required (simulate work)
	time.Sleep(time.Second * time.Duration(r.requestSleepSeconds))

	windowSizeAgo := now.Add(-time.Second * time.Duration(r.wSizeSeconds))

	newEarliestIdx := 0
	// lock as soon as we access r
	r.lock.Lock()
	for i, e := range r.window {
		if e.Before(windowSizeAgo) {
			continue
		}
		newEarliestIdx = i
		break
	}
	r.window = r.window[newEarliestIdx:]
	windowLength := len(r.window)
	r.window = append(r.window, now)
	r.lock.Unlock()

	return windowLength
}

// SaveCounter creates the file under path and truncates it (if it already existed) to fill it with the current window
func (r *RequestWindow) SaveCounter(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	err = os.Truncate(path, 0)
	if err != nil {
		return err
	}

	// if there is nothing to save, remove the file
	if r.window == nil || len(r.window) == 0 {
		err := os.Remove(path)
		if err != nil {
			return fmt.Errorf("unable to remove file: %w", err)
		}
	}

	for _, e := range r.window {
		_, err = f.WriteString(e.Format(requestWindowTimeFormat) + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}
