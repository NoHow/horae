package main

import (
	"fmt"
	"log"
	"strconv"
)

const (
	MENU_MAIN_MENU = iota
	MENU_INIT_FOCUS
	MENU_INIT_BREAK
	MENU_INFOCUS
	MENU_INBREAK
	MENU_SETTINGS
	MENU_SETTINGS_WORKDAY
	MENU_SETTINGS_FOCUS_DURATION
	MENU_SETTINGS_BREAK_DURATION
	MENU_SETTINGS_WORKDAY_TASK_EDIT
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

		env.timeKeepers[chatId] = startTimeKeeper(chatId, user.FocusDurationMins, env.onTimekeepStopped)
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

		env.timeKeepers[chatId] = startTimeKeeper(chatId, user.FocusDurationMins, env.onTimekeepStopped)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_STOP_BREAK),
			replyText:     fmt.Sprintf("Focus started! I will keep your time guard for %v minutes", user.FocusDurationMins),
			userAction:    UserAction{CurrentMenu: MENU_INBREAK},
		}
	case TTEXT_SETTINGS:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION, TTEXT_CHANGE_WORKDAY),
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
	case TTEXT_CHANGE_WORKDAY:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: generateWorkdaySettingsKeyboard(user.Workday.getTaskNames()),
			replyText:     "Choose task to edit",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY},
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
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION, TTEXT_CHANGE_WORKDAY),
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
			replyKeyboard: GenerateCustomKeyboard(TTEXT_FOCUS_DURATION, TTEXT_BREAK_DURATION, TTEXT_CHANGE_WORKDAY),
			replyText:     "Going back to the settings menu",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS},
		}
	}
	return
}

func processSettingsWorkdayMenu(messageText string, chatId ChatId, user User, users *Users) (result MenuProcessorResult, err error) {
	action := user.LastAction.Action
	context := user.LastAction.Context
	switch action {
	case SELECT_TASK_NAME_ACTION:
		tasks := user.Workday.getTaskNames()
		if len(tasks) >= MAX_TASKS_PER_DAY {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_TEXT,
				replyText:    "You can't add more tasks, please delete one before adding new",
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY},
			}, nil
		}

		newTaskName := messageText
		_, err := user.Workday.getTaskByName(newTaskName)
		if err == nil {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_TEXT,
				replyText:    "Task with this name already exists",
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY},
			}, nil
		}

		focusSessions := []string{"1", "2", "4", "6", "8"}
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(focusSessions...),
			replyText:     fmt.Sprintf("Createed task %v, please select how many focus periods do you want to spend on it?", newTaskName),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY, Action: SELECT_TASK_PERIODS_ACTION, Context: map[string]string{"taskName": newTaskName}},
		}
	case SELECT_TASK_PERIODS_ACTION:
		taskName, ok := context["taskName"]
		if !ok {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_KEYBOARD,
				replyText:    "I'm sorry, something went wrong, please try again",
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY},
			}, nil
		}

		taskPeriods, err := strconv.Atoi(messageText)
		var invalidInputMsg string
		if err != nil {
			invalidInputMsg = "Invalid input, please try again"
		} else if taskPeriods < 1 || taskPeriods > 9 {
			invalidInputMsg = "Number of periods should be greater than 0 and less than 10"
		}
		if invalidInputMsg != "" {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_KEYBOARD,
				replyText:    invalidInputMsg,
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY, Action: SELECT_TASK_PERIODS_ACTION, Context: map[string]string{"taskName": taskName}},
			}, nil
		}

		user.Workday.addTask(Task{
			Name:         taskName,
			FocusPeriods: taskPeriods,
		})
		err = users.updateUser(chatId, user)
		if err != nil {
			log.Printf("Error updating user when creating task: %v", err)
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_TEXT,
				replyText:    "I'm sorry, I couldn't find your user data, please type /start",
			}, nil
		}
		taskNames := user.Workday.getTaskNames()
		taskNames = append(taskNames, TTEXT_ADD_TASK)
		taskNames = append(taskNames, TTEXT_MAIN_MENU)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(taskNames...),
			replyText:     "Task created, please choose another task to edit or go back to main menu",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY},
		}
	}
	if action != INVALID_ACTION {
		return
	}

	switch messageText {
	case TTEXT_MAIN_MENU:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateMainKeyboard(),
			replyText:     "Main menu",
			userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
		}
	case TTEXT_ADD_TASK:
		result = MenuProcessorResult{
			responseType: RESPONSE_TYPE_TEXT,
			replyText:    "Please enter task name",
			userAction: UserAction{
				CurrentMenu: MENU_SETTINGS_WORKDAY,
				Action:      SELECT_TASK_NAME_ACTION,
			},
		}
	default:
		task, err := user.Workday.getTaskByName(messageText)
		if err != nil {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_KEYBOARD,
			}, fmt.Errorf("task with name - [%v] not found", messageText)
		}
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_EDIT_TASK_NAME, TTEXT_EDIT_TASK_PERIODS, TTEXT_EDIT_TASK_ORDER, TTEXT_DELETE_TASK),
			replyText:     fmt.Sprintf("Task: %v, Focus Periods: %v", task.Name, task.FocusPeriods),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Context: map[string]string{"taskName": task.Name}},
		}
	}
	return
}

func processSettingsWorkdayTaskEditMenu(messageText string, chatId ChatId, user User, users *Users) (result MenuProcessorResult, err error) {
	action := user.LastAction.Action
	taskName := user.LastAction.Context["taskName"]
	switch action {
	case SELECT_TASK_NAME_ACTION:
		newTaskName := messageText
		task, err := user.Workday.getTaskByName(newTaskName)
		if err == nil {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_TEXT,
				replyText:    "Task with this name already exists",
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Context: map[string]string{"taskName": taskName}},
			}, nil
		}

		user.Workday.updateTask(taskName, Task{
			Name:         newTaskName,
			FocusPeriods: task.FocusPeriods,
		})
		users.updateUser(chatId, user)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_EDIT_TASK_NAME, TTEXT_EDIT_TASK_PERIODS, TTEXT_EDIT_TASK_ORDER, TTEXT_DELETE_TASK, TTEXT_MAIN_MENU),
			replyText:     fmt.Sprintf("Updated task name to the %v, what do you want to do next?", newTaskName),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Context: map[string]string{"taskName": newTaskName}},
		}
	case SELECT_TASK_PERIODS_ACTION:
		taskPeriods, err := strconv.Atoi(messageText)
		var invalidInputMsg string
		if err != nil {
			invalidInputMsg = "Invalid input, please try again"
		} else if taskPeriods < 1 || taskPeriods > 9 {
			invalidInputMsg = "Number of periods should be greater than 0 and less than 10"
		}
		if invalidInputMsg != "" {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_KEYBOARD,
				replyText:    invalidInputMsg,
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY, Action: SELECT_TASK_PERIODS_ACTION, Context: map[string]string{"taskName": taskName}},
			}, nil
		}

		user.Workday.updateTask(taskName, Task{
			Name:         taskName,
			FocusPeriods: taskPeriods,
		})
		users.updateUser(chatId, user)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_EDIT_TASK_NAME, TTEXT_EDIT_TASK_PERIODS, TTEXT_EDIT_TASK_ORDER, TTEXT_DELETE_TASK, TTEXT_MAIN_MENU),
			replyText:     fmt.Sprintf("Updated task periods to the %v, what do you want to do next?", taskPeriods),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Context: map[string]string{"taskName": taskName}},
		}
	case SELECT_TASK_ORDER_ACTION:
		taskOrder, err := strconv.Atoi(messageText)
		taskNames := user.Workday.getTaskNames()
		var invalidInputMsg string
		if err != nil {
			invalidInputMsg = "Invalid input, please try again"
		} else if taskOrder < 1 || taskOrder > len(taskNames) {
			invalidInputMsg = fmt.Sprintf("Task order should be greater than 0 and less than %v", len(taskNames))
		}
		if invalidInputMsg != "" {
			return MenuProcessorResult{
				responseType: RESPONSE_TYPE_KEYBOARD,
				replyText:    invalidInputMsg,
				userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY, Action: SELECT_TASK_ORDER_ACTION, Context: map[string]string{"taskName": taskName}},
			}, nil
		}

		user.Workday.setTaskPosition(taskName, taskOrder-1)
		users.updateUser(chatId, user)
		updatedTaskQueue := generateTaskPositionsText(taskName, user.Workday.getTaskNames())
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(TTEXT_EDIT_TASK_NAME, TTEXT_EDIT_TASK_PERIODS, TTEXT_EDIT_TASK_ORDER, TTEXT_DELETE_TASK, TTEXT_MAIN_MENU),
			replyText:     fmt.Sprintf("Updated task order!\n %v Please select what do you want to do next?", updatedTaskQueue),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Context: map[string]string{"taskName": taskName}},
		}
	}
	if action != INVALID_ACTION {
		return
	}

	switch messageText {
	case TTEXT_EDIT_TASK_NAME:
		result = MenuProcessorResult{
			responseType: RESPONSE_TYPE_TEXT,
			replyText:    "Please enter new task name",
			userAction:   UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Action: SELECT_TASK_NAME_ACTION, Context: map[string]string{"taskName": taskName}},
		}
	case TTEXT_EDIT_TASK_PERIODS:
		focusSessions := []string{"1", "2", "4", "6", "8"}
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(focusSessions...),
			replyText:     "Please select number number of focus sessions for this task",
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Action: SELECT_TASK_PERIODS_ACTION, Context: map[string]string{"taskName": taskName}},
		}
	case TTEXT_EDIT_TASK_ORDER:
		taskNames := user.Workday.getTaskNames()
		positions := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
		message := "Please select new position for the task\n"
		message += generateTaskPositionsText(taskName, taskNames)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateCustomKeyboard(positions[0:len(taskNames)]...),
			replyText:     message,
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY_TASK_EDIT, Action: SELECT_TASK_ORDER_ACTION, Context: map[string]string{"taskName": taskName}},
		}
	case TTEXT_DELETE_TASK:
		user.Workday.deleteTask(taskName)
		users.updateUser(chatId, user)
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyText:     fmt.Sprintf("Task %v deleted", taskName),
			userAction:    UserAction{CurrentMenu: MENU_SETTINGS_WORKDAY},
			replyKeyboard: generateWorkdaySettingsKeyboard(user.Workday.getTaskNames()),
		}
	case TTEXT_MAIN_MENU:
		result = MenuProcessorResult{
			responseType:  RESPONSE_TYPE_KEYBOARD,
			replyKeyboard: GenerateMainKeyboard(),
			replyText:     "Main menu",
			userAction:    UserAction{CurrentMenu: MENU_MAIN_MENU},
		}
	}
	return
}

func generateTaskPositionsText(priorityTask string, taskNames []string) (message string) {
	for i, name := range taskNames {
		if name != priorityTask {
			message = message + fmt.Sprintf("%v. %v\n", i+1, name)
		} else {
			message = message + fmt.Sprintf("<b>%v. %v</b>\n", i+1, name)
		}
	}
	return message
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
