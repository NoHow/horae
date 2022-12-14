package main

import (
	"fmt"
	"sync"
	"time"
)

type TimeKeeper struct {
	secondsLeft int
	isStopped   bool
	stopMut     sync.Mutex
}

func startTimeKeeper(chatId ChatId, focusDuration int, finishMessage string, callback timeekeepStoppedCallback) *TimeKeeper {
	ticker := time.NewTicker(time.Second * 1)
	tk := TimeKeeper{
		secondsLeft: 0,
		isStopped:   false,
	}

	go tk.watchTime(chatId, focusDuration, finishMessage, ticker, callback)
	return &tk
}

func (tk *TimeKeeper) stopTimeKeep() bool {
	tk.stopMut.Lock()
	defer tk.stopMut.Unlock()
	if !tk.isStopped {
		tk.isStopped = true
		return true
	}
	return false
}

func (tk *TimeKeeper) watchTime(chatId ChatId, focusDuration int, finishMessage string, ticker *time.Ticker, callback timeekeepStoppedCallback) {
	tk.secondsLeft = focusDuration * 60
	for {
		if tk.isStopped {
			fmt.Println("TimeKeeper stopped")
			return
		}

		_ = <-ticker.C
		tk.secondsLeft = tk.secondsLeft - 1
		if tk.secondsLeft == 0 {
			ok := tk.stopTimeKeep()
			if ok {
				callback(chatId, finishMessage)
			}
		}
	}
}
