package main

import (
	"log"
	"strconv"
	"sync"
	"time"
)

var _backgroundFunc func()
var _backgroundTask sync.WaitGroup

func restartBackgroundTask(oldValue *string, newValue *string) {
	log.Println("Restart background task")
	if nil == _backgroundFunc {
		return
	}

	log.Println("Stopping background task")
	// wg := sync.WaitGroup{}
	// wg.Add(1)
	// go func() {
	// 	stopBackgroundTask()
	// 	wg.Done()
	// }()
	go stopBackgroundTask()

	log.Println("Waiting for background task to stop")

	// wg.Wait()
	_backgroundTask.Wait()
	log.Println("Restarting background task")
	startBackgroundTask(_backgroundFunc)
}

func stopBackgroundTask() {
	application.ticker.Stop()
	application.stopPoller <- true
}

func startBackgroundTask(f func()) {
	// rawPt, prs := getPreference("poll_time")
	rawPt, prs := application.preferences.Get("poll_time")
	if !prs {
		log.Fatalf("No poll_time preference defined (%s)", *rawPt)
	} else {
		pollTime, _ := strconv.Atoi(*rawPt)
		log.Printf("Starting background process, polling every %d seconds", pollTime)
		application.ticker = time.NewTicker(time.Duration(pollTime) * time.Second)
		application.stopPoller = make(chan bool)

		_backgroundFunc = f
		_backgroundTask = sync.WaitGroup{}
		_backgroundTask.Add(1)

		stopLoop := false

		go func() {
			log.Println("Entering background task loop")
			for {
				select {
				case <-application.stopPoller:
					stopLoop = true
					break
				case <-application.ticker.C:
					f()
				}

				if stopLoop {
					break
				}
			}
			log.Println("Exiting background task loop")
			_backgroundTask.Done()
		}()
	}
}
