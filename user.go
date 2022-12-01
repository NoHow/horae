package main

import (
	"fmt"
	"log"
	"sync"
)

type ChatId int64

type UserAction struct {
	CurrentMenu int               `json:"current_menu"`
	Action      int               `json:"menu_option"`
	Context     map[string]string `json:"context"`
}

type User struct {
	FirstName         string     `json:"first_name"`
	FocusDurationMins int        `json:"focus_duration"`
	BreakDurationMins int        `json:"break_duration"`
	LastAction        UserAction `json:"last_action,omitempty"`
}

type Users struct {
	data map[ChatId]User
	mut  sync.Mutex
}

func (u *User) setFocusDuration(duration int) error {
	switch duration {
	case 15:
		u.FocusDurationMins = 15
	case 30:
		u.FocusDurationMins = 30
	case 45:
		u.FocusDurationMins = 45
	case 60:
		u.FocusDurationMins = 60
	default:
		return fmt.Errorf("invalid focus duration [%v]", duration)
	}
	return nil
}

func (u *User) setBreakDuration(duration int) error {
	switch duration {
	case 5:
		u.BreakDurationMins = 5
	case 10:
		u.BreakDurationMins = 10
	case 15:
		u.BreakDurationMins = 15
	case 20:
		u.BreakDurationMins = 20
	default:
		return fmt.Errorf("invalid break duration [%v]", duration)
	}
	return nil
}

func (u *User) getActionContextField(field string) (string, error) {
	if u.LastAction.Context == nil {
		return "", fmt.Errorf("user action context is nil")
	}
	if value, ok := u.LastAction.Context[field]; ok {
		return value, nil
	}
	return "", fmt.Errorf("field [%v] not found in user action context", field)
}

func (u *Users) add(chatId ChatId, user User) (result bool) {
	u.mut.Lock()
	if _, ok := u.data[chatId]; ok {
		log.Printf("User with chat id [%v] already exists", chatId)
		result = false
	} else {
		u.data[chatId] = user
		log.Printf("user with chat id [%v] added to the list", chatId)
		result = true
	}
	u.mut.Unlock()
	return
}

func (u *Users) updateUser(chatId ChatId, user User) error {
	u.mut.Lock()
	if _, ok := u.data[chatId]; ok {
		u.data[chatId] = user
	} else {
		return fmt.Errorf("user with chat id [%v] not found", chatId)
	}
	u.mut.Unlock()
	return nil
}

func (u *Users) saveLastUserAction(chatId ChatId, action UserAction) {
	u.mut.Lock()
	if user, ok := u.data[chatId]; ok {
		user.LastAction = action
		u.data[chatId] = user
	} else {
		log.Printf("user with chat id [%v] not found", chatId)
	}
	u.mut.Unlock()
}
