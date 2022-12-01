package main

import (
	"fmt"
)

const (
	MENU_MAIN_MENU = iota
	MENU_INIT_FOCUS
	MENU_INIT_BREAK
	MENU_INFOCUS
	MENU_INBREAK
	MENU_SETTINGS
	MENU_SETTINGS_FOCUS_DURATION
	MENU_SETTINGS_BREAK_DURATION
)

const (
	RESPONSE_TYPE_NONE = iota
	RESPONSE_TYPE_TEXT
	RESPONSE_TYPE_KEYBOARD
)

type MenuProcessorResult struct {
	responseType  int
	replyKeyboard TReplyKeyboard
	replyText     string
	userAction    UserAction
}

func processMainMenu(messageText string, user User, chatId ChatId, env *environment) (result MenuProcessorResult, err error) {
	switch messageText {
	case TTEXT_START_FOCUS:
		_, ok := env.timeKeepers[chatId]
		if ok {
			//return true, fmt.Sprintf("user with chat id - [%v] already has a time keeper", chatId)
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_STOP_FOCUS),
				replyText:     "Oops, looks like you already have an active time guard!",
				userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
			}, nil
		}

		env.timeKeepers[chatId] = startTimeKeeper(chatId, user.FocusDurationMins, "The focus session ended, you can rest now!", env.onTimekeepStopped)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_STOP_FOCUS),
			replyText:     fmt.Sprintf("Focus started! I will keep your time guard for %v minutes", user.FocusDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
		}
	case TTEXT_START_BREAK:
		_, ok := env.timeKeepers[chatId]
		if ok {
			//return true, fmt.Sprintf("user with chat id - [%v] already has a time keeper", chatId)
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateMainKeyboard(),
				replyText:     "Oops, looks like you already have an active time guard!",
				userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
			}, fmt.Errorf("user with chat id - [%v] already has a time keeper", chatId)
		}

		env.timeKeepers[chatId] = startTimeKeeper(chatId, user.BreakDurationMins, "Break is over. Let's get back to work!", env.onTimekeepStopped)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_STOP_BREAK),
			replyText:     fmt.Sprintf("Focus started! I will keep your time guard for %v minutes", user.BreakDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_INBREAK},
		}
	case TTEXT_SETTINGS:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION),
			replyText:     "Settings",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS},
		}
	}
	return
}

func processInFocusMenu(messageText string, chatId ChatId, timeKeepers *map[ChatId]*TimeKeeper) (result MenuProcessorResult, err error) {
	switch messageText {
	case TTEXT_STOP_FOCUS:
		tk, ok := (*timeKeepers)[chatId]
		if !ok {
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_STOP_FOCUS),
				replyText:     "Oops, looks like you don't have an active time guard!",
				userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
			}, nil
		} else {
			ok = tk.stopTimeKeep()
			if !ok {
				return MenuProcessorResult{
					responseType: RESPONSE_TYPE_NONE,
				}, nil
			} else {
				delete(*timeKeepers, chatId)
				result = MenuProcessorResult{
					responseType:  RESPONSE_TYPE_KEYBOARD,
					replyKeyboard: GenerateMainKeyboard(),
					replyText:     "Focus stopped",
					userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
				}
			}
		}

	}
	return
}

func processInBreakMenu(messageText string, chatId ChatId, timeKeepers *map[ChatId]*TimeKeeper) (result MenuProcessorResult, err error) {
	switch messageText {
	case TTEXT_STOP_BREAK:
		tk, ok := (*timeKeepers)[chatId]
		if !ok {
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_STOP_BREAK),
				replyText:     "Oops, looks like you don't have an active time guard!",
				userAction:    UserAction{CurrentMenu: MENU_INBREAK},
			}, nil
		} else {
			ok = tk.stopTimeKeep()
			if !ok {
				return MenuProcessorResult{
					responseType: RESPONSE_TYPE_NONE,
				}, nil
			} else {
				delete(*timeKeepers, chatId)
				result = MenuProcessorResult{
					responseType:  RESPONSE_TYPE_KEYBOARD,
					replyKeyboard: GenerateMainKeyboard(),
					replyText:     "Break stopped",
					userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
				}
			}
		}

	}
	return
}

func processSettingsMenu(messageText string, user User) (result MenuProcessorResult, err error) {
	switch messageText {
	case TTEXT_FOCUS_DURATION:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_CHANGE_FOCUS_DURATION, TTEXT_BACK),
			replyText:     fmt.Sprintf("Your current focus duration is %v minutes. Do you want to change it?", user.FocusDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_FOCUS_DURATION},
		}
	case TTEXT_BREAK_DURATION:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_CHANGE_BREAK_DURATION, TTEXT_BACK),
			replyText:     fmt.Sprintf("Your current break duration is %v minutes. Do you want to change it?", user.BreakDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_BREAK_DURATION},
		}
	}
	return
}

func processSettingsFocusDurationMenu(messageText string, chatId ChatId, user User, users *Users, possibleDurations []string) (result MenuProcessorResult, err error) {
	switch user.LastAction.Action {
	case CHANGE_FOCUS_DURATION_ACTION:
		if messageText == "1 hour" {
			user.setFocusDuration(60)
		}

		index := findStringInSlice(possibleDurations, messageText)
		if index == -1 {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_TEXT,
				replyText:    "Oops, looks like you have entered wrong value. Please try again",
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_FOCUS_DURATION},
			}, nil
		}

		user.setFocusDuration((index + 1) * 15)
		users.updateUser(chatId, user)

		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateMainKeyboard(),
			replyText:     fmt.Sprintf("Your focus duration is now %v minutes! Going back to the main menu", user.FocusDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
		}
	}

	switch messageText {
	case TTEXT_CHANGE_FOCUS_DURATION:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(possibleDurations...),
			replyText:     "Choose new focus duration",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_FOCUS_DURATION, Action: CHANGE_FOCUS_DURATION_ACTION},
		}
	case TTEXT_BACK:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION),
			replyText:     "Going back to the settings menu",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS},
		}
	}
	return
}

func processSettingsBreakDurationMenu(messageText string, chatId ChatId, user User, users *Users, possibleDurations []string) (result MenuProcessorResult, err error) {
	switch user.LastAction.Action {
	case CHANGE_BREAK_DURATION_ACTION:
		index := findStringInSlice(possibleDurations, messageText)
		if index == -1 {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_TEXT,
				replyText:    "Oops, looks like you have entered wrong value. Please try again",
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_BREAK_DURATION},
			}, nil
		}

		user.setBreakDuration((index + 1) * 5)
		users.updateUser(chatId, user)

		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateMainKeyboard(),
			replyText:     fmt.Sprintf("Your break duration is now %v minutes! Going back to the main menu", user.BreakDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
		}
	}

	switch messageText {
	case TTEXT_CHANGE_BREAK_DURATION:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(possibleDurations...),
			replyText:     "Choose new break duration",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_BREAK_DURATION, Action: CHANGE_BREAK_DURATION_ACTION},
		}
	case TTEXT_BACK:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION),
			replyText:     "Going back to the settings menu",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS},
		}
	}
	return
}

func processInitFocusMenu(messageText string, id ChatId, user User, users *Users, focusDurations []string, pauseDurations []string) (result MenuProcessorResult, err error) {
	if messageText == "1 hour" {
		user.setFocusDuration(60)
	} else {
		index := findStringInSlice(focusDurations[0:3], messageText)
		if index == -1 {
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(focusDurations...),
				replyText:     "Sorry, I didn't get that. Please select one of the options below",
				userAction:    UserAction{CurrentMenu: MENU_INIT_FOCUS},
			}, nil
		}

		user.setFocusDuration((index + 1) * 15)
	}
	users.updateUser(id, user)

	result = MenuProcessorResult{
		responseType:  RESPONSE_TYPE_KEYBOARD,
		replyKeyboard: GenerateCustomKeyboard(pauseDurations...),
		replyText:     "Great! Now select your break duration",
		userAction:    UserAction{CurrentMenu: MENU_INIT_BREAK},
	}
	return
}

func processInitBreakMenu(messageText string, id ChatId, user User, users *Users, pauseDurations []string) (result MenuProcessorResult, err error) {
	index := findStringInSlice(pauseDurations[:], messageText)
	if index == -1 {
		return MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(pauseDurations...),
			replyText:     "Sorry, I didn't get that. Please select one of the options below",
			userAction:    UserAction{CurrentMenu: MENU_INIT_BREAK},
		}, nil
	}

	user.setBreakDuration((index + 1) * 5)
	users.updateUser(id, user)

	result = MenuProcessorResult{
		responseType:  RESPONSE_TYPE_KEYBOARD,
		replyKeyboard: GenerateMainKeyboard(),
		replyText:     "Great! Now you all set to start your first focus session",
		userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
	}
	return
}
