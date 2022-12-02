package main

import (
	"fmt"
	"log"
)

const (
	TTEXT_START_COMMAND     = "/start"
	TTEXT_DURATIONS_COMMAND = "/durations"
	TTEXT_MAIN_MENU_COMMAND = "/main"

	TTEXT_MAIN_MENU             = "Main menu"
	TTEXT_START_FOCUS           = "Let's focus " + EMOJI_SEEDLING
	TTEXT_SETTINGS              = "Settings " + EMOJI_WRENCH
	TTEXT_STOP_FOCUS            = "Stop focus " + EMOJI_FALLEN_LEAF
	TTEXT_TIME_LEFT_FOCUS       = "Time left " + EMOJI_HERB
	TTEXT_TIME_LEFT_BREAK       = "Time left " + EMOJI_PERSON_IN_LOTUS_POSITION
	TTEXT_START_BREAK           = "Let's take a break " + EMOJI_PERSON_HOT_BEVERAGE
	TTEXT_STOP_BREAK            = "Stop break " + EMOJI_PERSON_RUNNING
	TTEXT_FOCUS_DURATION        = "Focus duration"
	TTEXT_BREAK_DURATION        = "Break duration"
	TTEXT_CHANGE_FOCUS_DURATION = "Change focus duration"
	TTEXT_CHANGE_BREAK_DURATION = "Change break duration"
	TTEXT_BACK                  = EMOJI_BACK

	EMOJI_SEEDLING                  = "\U0001F331"
	EMOJI_HERB                      = "\U0001F33F"
	EMOJI_FALLEN_LEAF               = "\U0001F342"
	EMOJI_PERSON_HOT_BEVERAGE       = "\u2615"
	EMOJI_WRENCH                    = "\U0001F527"
	EMOJI_STOPWATCH                 = "\u23F1"
	EMOJI_CROSS_MARK                = "\u274C"
	EMOJI_BACK                      = "\U0001F519"
	EMOJI_WHITE_MEDIUM_SMALL_SQUARE = "\u25FD"
	EMOJI_PERSON_IN_LOTUS_POSITION  = "\U0001F9D8"
	EMOJI_PERSON_RUNNING            = "\U0001F3C3"
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
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_FOCUS, TTEXT_STOP_FOCUS),
				replyText:     "Oops, looks like you already have an active time guard!",
				userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
			}, nil
		}

		env.timeKeepers[chatId] = startTimeKeeper(chatId, user.FocusDurationMins, "The focus session ended, you can rest now!", env.onTimekeepStopped)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_FOCUS, TTEXT_STOP_FOCUS),
			replyText:     fmt.Sprintf("Focus started! I will keep you focused for %v minutes", user.FocusDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
		}
	case TTEXT_START_BREAK:
		_, ok := env.timeKeepers[chatId]
		if ok {
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_BREAK, TTEXT_STOP_BREAK),
				replyText:     "Oops, looks like you already have an active time guard!",
				userAction:    UserAction{CurrentMenu: MENU_INBREAK},
			}, fmt.Errorf("user with chat id - [%v] already has a time keeper", chatId)
		}

		env.timeKeepers[chatId] = startTimeKeeper(chatId, user.BreakDurationMins, "Break is over. Let's get back to work!", env.onTimekeepStopped)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_BREAK, TTEXT_STOP_BREAK),
			replyText:     fmt.Sprintf("Break started! You can rest for %v minutes", user.BreakDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_INBREAK},
		}
	case TTEXT_SETTINGS:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION, TTEXT_MAIN_MENU),
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
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_FOCUS, TTEXT_STOP_FOCUS),
				replyText:     "Oops, looks like you don't have an active focus!",
				userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
			}, nil
		} else {
			ok = tk.stopTimeKeep()
			if !ok {
				log.Printf("Failed to stop timekeeper for chat id - [%v]", chatId)
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
	case TTEXT_TIME_LEFT_FOCUS:
		tk, ok := (*timeKeepers)[chatId]
		if !ok {
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_FOCUS, TTEXT_STOP_FOCUS),
				replyText:     "Oops, looks like you don't have an active focus!",
				userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
			}, nil
		} else {
			result = MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_FOCUS, TTEXT_STOP_FOCUS),
				replyText:     fmt.Sprintf("You have %v to go", generateTimeLeftString(tk)),
				userAction:    UserAction{CurrentMenu: MENU_INFOCUS},
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
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_BREAK, TTEXT_STOP_BREAK),
				replyText:     "Oops, looks like you don't have an active break!",
				userAction:    UserAction{CurrentMenu: MENU_INBREAK},
			}, nil
		} else {
			ok = tk.stopTimeKeep()
			if !ok {
				log.Printf("Failed to stop timekeeper for chat id - [%v]", chatId)
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
	case TTEXT_TIME_LEFT_BREAK:
		tk, ok := (*timeKeepers)[chatId]
		if !ok {
			return MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_BREAK, TTEXT_STOP_BREAK),
				replyText:     "Oops, looks like you don't have an active break!",
				userAction:    UserAction{CurrentMenu: MENU_INBREAK},
			}, nil
		} else {
			result = MenuProcessorResult{
				responseType:  RESPONSE_TYPE_KEYBOARD,
				replyKeyboard: GenerateCustomKeyboard(TTEXT_TIME_LEFT_BREAK, TTEXT_STOP_BREAK),
				replyText:     fmt.Sprintf("You can still relax for %v", generateTimeLeftString(tk)),
				userAction:    UserAction{CurrentMenu: MENU_INBREAK},
			}
		}
	}
	return
}

func generateTimeLeftString(tk *TimeKeeper) string {
	if tk.secondsLeft > 0 && tk.secondsLeft%60 == 0 {
		return fmt.Sprintf("<b>%v minutes</b>", tk.secondsLeft/60)
	} else if tk.secondsLeft/60 == 0 {
		return fmt.Sprintf("<b>%v seconds</b>", tk.secondsLeft)
	} else {
		return fmt.Sprintf("<b>%v minutes and %v seconds</b>", tk.secondsLeft/60, tk.secondsLeft%60)
	}
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
	case TTEXT_MAIN_MENU:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateMainKeyboard(),
			replyText:     "Back to main menu",
			userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
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
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION, TTEXT_MAIN_MENU),
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
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION, TTEXT_MAIN_MENU),
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
