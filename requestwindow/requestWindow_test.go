package requestwindow

import (
	"bufio"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func setupTestWindowFile(path string, timestamp string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	err = os.Truncate(path, 0)
	if err != nil {
		return err
	}

	_, err = f.WriteString(timestamp + "\n")
	if err != nil {
		return err
	}

	return nil
}

func cleanupTestWindowFile(path string) error {
	return os.Remove(path)
}

func TestNewWindow(t *testing.T) {
	now := time.Now()
	tests := []struct {
		args []time.Time
		name string
		want *RequestWindow
	}{
		{
			name: "Create empty window",
			want: &RequestWindow{},
		},
		{
			args: []time.Time{now},
			name: "Creates window with content",
			want: &RequestWindow{window: []time.Time{now}},
		},
	}
	for _, tt := range tests {
		rw := NewWindow(tt.args, 60, 2)
		if !reflect.DeepEqual(rw.window, tt.want.window) {
			t.Errorf("want: %+v, got: %+v for %s\n", tt.want.window, rw.window, tt.name)
		}
	}
}

func TestNewWindowFromFile(t *testing.T) {
	filePath := "./test"
	filePath2 := "./test2"
	now := time.Now()
	nowParsedWithFormat, err := time.ParseInLocation(requestWindowTimeFormat, now.Format(requestWindowTimeFormat), time.Local)
	if err != nil {
		t.Errorf("could not setup test: %s", err.Error())
	}
	err = setupTestWindowFile(filePath, now.Format(requestWindowTimeFormat))
	if err != nil {
		t.Errorf("could not setup test: %s", err.Error())
	}
	err = setupTestWindowFile(filePath2, "")
	if err != nil {
		t.Errorf("could not setup test: %s", err.Error())
	}
	defer func() {
		cleanupTestWindowFile(filePath)

		cleanupTestWindowFile(filePath2)
	}()

	tests := []struct {
		args    string
		name    string
		isError bool
		want    *RequestWindow
	}{
		{
			args:    filePath,
			name:    "Create window from file",
			isError: false,
			want:    &RequestWindow{window: []time.Time{nowParsedWithFormat}},
		},
		{
			args:    "",
			isError: true,
			name:    "Do not pass a path",
		},
		{
			args:    filePath2,
			isError: true,
			name:    "Empty file",
		},
	}
	for _, tt := range tests {
		rw, err := NewWindowFromFile(tt.args, 60, 2)
		if err != nil && !tt.isError {
			t.Errorf("got unexpected error: %s", err.Error())
		}
		if !tt.isError && !reflect.DeepEqual(rw.window, tt.want.window) {
			t.Errorf("want: %+v, got: %+v for %s\n", tt.want.window, rw.window, tt.name)
		}
	}
}

func TestSaveCounter(t *testing.T) {
	path := "./test"
	now := time.Now()

	defer func() {
		cleanupTestWindowFile(path)
	}()

	tests := []struct {
		rw         *RequestWindow
		args       string
		name       string
		isError    bool
		fileExists bool
	}{
		{
			rw:         &RequestWindow{window: []time.Time{now}},
			args:       path,
			name:       "Save Window with content",
			isError:    false,
			fileExists: true,
		},
		{
			rw:         &RequestWindow{},
			args:       path,
			name:       "Save Window without content",
			isError:    false,
			fileExists: false,
		},
		{
			rw:      &RequestWindow{window: []time.Time{now}},
			args:    "&/F&§",
			name:    "Incorrect path",
			isError: true,
		},
	}
	for _, tt := range tests {
		err := tt.rw.SaveCounter(tt.args)
		if err != nil && !tt.isError {
			t.Errorf("unexpected error: %s", err.Error())
		}
		if tt.fileExists {
			if _, statErr := os.Stat(tt.args); errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("file was not created")
			}

			f, err := os.Open(path)
			if err != nil {
				t.Errorf("could not open file for verification: %s", err.Error())
			}

			scanner := bufio.NewScanner(f)
			scanner.Scan()
			if scanner.Text() != tt.rw.window[0].Format(requestWindowTimeFormat) {
				t.Errorf("file content was not written correctly")
			}
		}

	}
}

// GetCounter test will be split into separate functions to enhance readability
func TestGetCounterCorrectCounter(t *testing.T) {
	rw := RequestWindow{wSizeSeconds: 5}
	first := rw.GetCounter()
	second := rw.GetCounter()
	third := rw.GetCounter()

	if first != 0 || second != 1 || third != 2 {
		t.Errorf("counter is not incrementing correctly: %d, %d, %d", first, second, third)
	}
}

func TestGetCounterMovingWindow(t *testing.T) {
	rw := RequestWindow{wSizeSeconds: 5}
	first := rw.GetCounter()
	time.Sleep(time.Second * 3)
	second := rw.GetCounter()
	time.Sleep(time.Second * 3)
	third := rw.GetCounter()
	// f = first; s = second; t = third; - = sleep; every symbol a second
	//fffff
	//---sssss
	//------ttttt
	if first != 0 || second != 1 || third != 1 {
		t.Errorf("window did not move correctly: %d, %d, %d", first, second, third)
	}
}

// Concurrency is hard to test, parallelism is even harder this test is not equivalent to real world behaviour
func TestGetCounterConcurrencyCorrectCounter(t *testing.T) {
	rw := RequestWindow{wSizeSeconds: 5}
	for i := 0; i < 100; i++ {
		go func() {
			rw.GetCounter()
		}()
	}

	// 5 seconds should be enough for 100 requests
	time.Sleep(time.Second * 5)
	crtCounter := rw.GetCounter()
	if crtCounter != 100 {
		t.Errorf("counter is not incrementing correctly: %d", crtCounter)
	}
}

func TestGetCounterSleepDurations(t *testing.T) {
	tests := []struct {
		rw      *RequestWindow
		runtime int
		name    string
	}{
		{
			rw:      &RequestWindow{wSizeSeconds: 5, requestSleepSeconds: 1},
			runtime: 1,
			name:    "one seconds of work",
		},
		{
			rw:      &RequestWindow{wSizeSeconds: 5, requestSleepSeconds: 2},
			runtime: 2,
			name:    "two seconds of work",
		},
		{
			rw:      &RequestWindow{wSizeSeconds: 5, requestSleepSeconds: 4},
			runtime: 4,
			name:    "four seconds of work",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_ = tt.rw.GetCounter()
			timeTaken := time.Now().Sub(start).Seconds()
			if timeTaken < float64(tt.runtime) || timeTaken > float64(tt.runtime)+1.0 {
				t.Errorf("runtime was %f, should have been %f\n", timeTaken, float64(tt.runtime))
			}
		})
	}
}
