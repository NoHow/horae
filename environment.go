package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

var validPath = regexp.MustCompile("^/(update)/+")
var reIpAddress = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`)

type environment struct {
	client      http.Client
	botKey      string
	ipAddress   string
	db          *hDataBase
	users       Users
	timeKeepers map[ChatId]*TimeKeeper
}

type TChat struct {
	Id   int64  `json:"id"`
	Type string `json:"type"`
}

type TKeyBoardButton struct {
	Text string `json:"text"`
}

type TReplyKeyboard struct {
	Keyboard        [][]TKeyBoardButton `json:"keyboard"`
	ResizeKeyboard  bool                `json:"resize_keyboard"`
	OneTimeKeyboard bool                `json:"one_time_keyboard"`
}

type TMessage struct {
	MessageId int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      TChat  `json:"chat"`
	From      TUser  `json:"from"`
}

type TMessageSend struct {
	ChatId ChatId `json:"chat_id"`
	Text   string `json:"text"`
}

type TKeyboardMessageSend struct {
	ChatId         ChatId         `json:"chat_id"`
	Text           string         `json:"text"`
	KeyboardMarkup TReplyKeyboard `json:"reply_markup"`
}

type TUser struct {
	Id        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type TUpdate struct {
	UpdateId int      `json:"update_id"`
	Message  TMessage `json:"message"`
}

func (u *TUpdate) GetChatId() ChatId {
	return ChatId(u.Message.Chat.Id)
}

const (
	TTEXT_START     = "/start"
	TTEXT_DURATIONS = "/durations"
	TTEXT_MAINMENU  = "/main"

	TTEXT_START_FOCUS           = "Let's focus"
	TTEXT_SETTINGS              = "Settings"
	TTEXT_STOP_FOCUS            = "Stop focus"
	TTEXT_TIME_LEFT             = "Time left"
	TTEXT_START_BREAK           = "Let's take a break"
	TTEXT_STOP_BREAK            = "Stop break"
	TTEXT_CHANGE_FOCUS_DURATION = "Focus duration"
	TTEXT_CHANGE_BREAK_DURATION = "Break duration"
	TTEXT_CHANGE_WORKDAY        = "Workday"
	TTEXT_ADD_TASK              = "Add task"
)

const (
	START_ACTION = iota
	MAIN_MENU_ACTION
	FOCUS_SELECTION_ACTION
	BREAK_SELECTION_ACTION
	FOCUS_ACTIVATION_ACTION
	BREAK_ACTIVATION_ACTION
	FOCUS_STOP_ACTION
	BREAK_STOP_ACTION
	SETTINGS_ACTION
	CHANGE_WORKDAY_ACTION
	TASKNAME_SELECTION_ACTION
	TASK_PERIODS_SELECTION_ACTION
)

func getPathValue(r *http.Request, pathCheck *regexp.Regexp) (string, error) {
	m := pathCheck.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return "", fmt.Errorf("url path is not valid")
	}

	log.Printf("getPathValue will return %v", m[1])
	return m[1], nil
}

func (env *environment) updateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("updateHandler")
	pageTitle, err := getPathValue(r, validPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Received page title - [%v]", pageTitle)
}

func (env *environment) rootHandler(w http.ResponseWriter, r *http.Request) {
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	Update := &TUpdate{}
	err = json.Unmarshal(buf, Update)
	if err != nil {
		log.Println(err)
		return
	}
	if Update.Message.Chat.Id <= 0 || Update.Message.From.Id <= 0 {
		log.Printf("invalid chat id - [%v] or user id - [%v]", Update.Message.Chat.Id, Update.Message.From.Id)
		return
	}

	keyboardMsg := TKeyboardMessageSend{
		KeyboardMarkup: TReplyKeyboard{
			OneTimeKeyboard: false,
		},
	}
	Msg := TMessageSend{}
	sendKeyboard := true
	focusDurations := []string{"15 minutes", "30 minutes", "45 minutes", "1 hour"}
	pauseDurations := []string{"5 minutes", "10 minutes", "15 minutes", "30 minutes"}
	switch Update.Message.Text {
	case TTEXT_START:
		newUser := User{
			FirstName: Update.Message.From.FirstName,
		}
		isNewUser := env.users.add(Update.GetChatId(), newUser)
		if isNewUser {
			env.db.saveUserData(Update.GetChatId(), newUser)
			msgText := fmt.Sprintf("Hello %s! I will help you to keep organised with your time!\n"+
				"Please select how long you want your focus duration to be?", Update.Message.From.FirstName)

			keyboardMsg = TKeyboardMessageSend{
				ChatId:         Update.GetChatId(),
				Text:           msgText,
				KeyboardMarkup: GenerateListKeyboard(focusDurations),
			}
		}
		env.users.saveLastUserAction(Update.GetChatId(), UserAction{
			Action:  START_ACTION,
			Context: nil,
		})
	case TTEXT_DURATIONS:
		user, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}

		msgText := fmt.Sprintf("Your focus duration is %v and your break duration is %v minutes", user.FocusDurationMins, user.BreakDurationMins)
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           msgText,
			KeyboardMarkup: GenerateMainKeyboard(),
		}
	case TTEXT_MAINMENU:
		_, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}
		msgText := "What would you like to do?"
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           msgText,
			KeyboardMarkup: GenerateMainKeyboard(),
		}
		env.users.saveLastUserAction(Update.GetChatId(), UserAction{
			Action:  MAIN_MENU_ACTION,
			Context: nil,
		})
	case TTEXT_START_FOCUS:
		user, err := env.startKeeperHelper(Update.GetChatId(), FOCUS_ACTIVATION_ACTION)
		if err != nil {
			log.Println(err)
			return
		}

		msgText := fmt.Sprintf("Focus session started! You have %v minutes to focus", user.FocusDurationMins)
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           msgText,
			KeyboardMarkup: GenerateCustomKeyboard(TTEXT_STOP_FOCUS),
		}
	case TTEXT_STOP_FOCUS:
		msgPtr, err := env.stopKeeperHelper(Update.GetChatId(), fmt.Sprintf("Focus session ended! You can take a break now"), FOCUS_STOP_ACTION)
		if err != nil {
			log.Println(err)
			return
		}
		keyboardMsg = *msgPtr
	case TTEXT_START_BREAK:
		user, err := env.startKeeperHelper(Update.GetChatId(), BREAK_ACTIVATION_ACTION)
		if err != nil {
			log.Println(err)
			return
		}

		msgText := fmt.Sprintf("Break session started! See you in %v minutes", user.BreakDurationMins)
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           msgText,
			KeyboardMarkup: GenerateCustomKeyboard(TTEXT_STOP_BREAK),
		}
	case TTEXT_STOP_BREAK:
		msgPtr, err := env.stopKeeperHelper(Update.GetChatId(), fmt.Sprintf("Break is over, time to get back to work!"), BREAK_STOP_ACTION)
		if err != nil {
			log.Println(err)
			return
		}
		keyboardMsg = *msgPtr
	case TTEXT_SETTINGS:
		_, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}

		msgText := fmt.Sprintf("What do you want to change?")
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           msgText,
			KeyboardMarkup: GenerateCustomKeyboard(TTEXT_CHANGE_FOCUS_DURATION, TTEXT_CHANGE_BREAK_DURATION, TTEXT_CHANGE_WORKDAY),
		}
		env.users.saveLastUserAction(Update.GetChatId(), UserAction{
			Action:  SETTINGS_ACTION,
			Context: nil,
		})
	case TTEXT_CHANGE_WORKDAY:
		user, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}
		taskNames := user.Workday.getTaskNames()
		taskNames = append(taskNames, TTEXT_ADD_TASK)
		msgText := fmt.Sprintf("What tasks do you want to modify?")
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           msgText,
			KeyboardMarkup: GenerateListKeyboard(taskNames),
		}
		keyboardMsg.KeyboardMarkup.OneTimeKeyboard = true
		env.users.saveLastUserAction(Update.GetChatId(), UserAction{
			Action:  CHANGE_WORKDAY_ACTION,
			Context: nil,
		})
	case TTEXT_ADD_TASK:
		_, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}

		msgText := fmt.Sprintf("What kind of task do you want to add?")
		Msg = TMessageSend{
			ChatId: Update.GetChatId(),
			Text:   msgText,
		}
		sendKeyboard = false
		env.users.saveLastUserAction(Update.GetChatId(), UserAction{
			Action:  TASKNAME_SELECTION_ACTION,
			Context: nil,
		})
	default:
		user, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}

		//process user input based on last action and context
		switch user.LastAction.Action {
		case START_ACTION:
			if Update.Message.Text == "1 hour" {
				user.setFocusDuration(60)
			}

			index := findStringInSlice(focusDurations[0:3], Update.Message.Text)
			if index == -1 {
				log.Printf("invalid focus duration - [%v]", Update.Message.Text)
				return
			}
			user.setFocusDuration((index + 1) * 15)
			env.users.data[Update.GetChatId()] = user
			env.users.saveLastUserAction(Update.GetChatId(), UserAction{
				Action:  FOCUS_SELECTION_ACTION,
				Context: nil,
			})
			env.db.saveUserData(Update.GetChatId(), user)

			msgText := fmt.Sprintf("Please select how long you want your breaks to be?")
			keyboardMsg = TKeyboardMessageSend{
				ChatId:         Update.GetChatId(),
				Text:           msgText,
				KeyboardMarkup: GenerateListKeyboard(pauseDurations),
			}
		case FOCUS_SELECTION_ACTION:
			if Update.Message.Text == "30 minutes" {
				user.setBreakDuration(30)
			}

			index := findStringInSlice(pauseDurations[0:3], Update.Message.Text)
			if index == -1 {
				log.Printf("invalid pause duration - [%v]", Update.Message.Text)
				return
			}
			user.setBreakDuration((index + 1) * 5)
			env.users.data[Update.GetChatId()] = user
			env.users.saveLastUserAction(Update.GetChatId(), UserAction{
				Action:  BREAK_SELECTION_ACTION,
				Context: nil,
			})
			env.db.saveUserData(Update.GetChatId(), user)

			msgText := fmt.Sprintf("Great! You are all set to start your first focus session!")
			keyboardMsg = TKeyboardMessageSend{
				ChatId:         Update.GetChatId(),
				Text:           msgText,
				KeyboardMarkup: GenerateMainKeyboard(),
			}
		default:
			msgText := fmt.Sprintf("Sorry, I don't understand you. Please select one of the options below")
			keyboardMsg = TKeyboardMessageSend{
				ChatId: Update.GetChatId(),
				Text:   msgText,
			}

		}
	}

	if sendKeyboard {
		env.marshalAndSendMessage(keyboardMsg)
	} else {
		env.marshalAndSendMessage(Msg)
	}
	log.Printf("Successfully processed message from user - [%v]", Update.Message.From.FirstName)
}

func (env *environment) startKeeperHelper(chatId ChatId, lastAction int) (User, error) {
	user, ok := env.users.data[chatId]
	if !ok {
		return user, fmt.Errorf("user with chat id - [%v] is not found", chatId)
	}
	_, ok = env.timeKeepers[chatId]
	if ok {
		return user, fmt.Errorf("user with chat id - [%v] already has a time keeper", chatId)
	}

	env.timeKeepers[chatId] = startTimeKeeper(chatId, user.FocusDurationMins, env.onTimekeepStopped)
	env.users.saveLastUserAction(chatId, UserAction{
		Action:  lastAction,
		Context: nil,
	})
	return user, nil
}

func (env *environment) stopKeeperHelper(chatId ChatId, successText string, lastAction int) (*TKeyboardMessageSend, error) {
	_, ok := env.users.data[chatId]
	if !ok {
		return nil, fmt.Errorf("user with chat id - [%v] is not found", chatId)
	}

	tk, ok := env.timeKeepers[chatId]
	if !ok {
		msgText := fmt.Sprintf("You don't have an active focus nor break session")
		msg := TKeyboardMessageSend{
			ChatId:         chatId,
			Text:           msgText,
			KeyboardMarkup: GenerateMainKeyboard(),
		}

		return &msg, nil
	} else {
		ok = tk.stopTimeKeep()
		if !ok {
			return &TKeyboardMessageSend{}, fmt.Errorf("failed to stop time keeper for user with chat id because it was already stopped - [%v]", chatId)
		}
		delete(env.timeKeepers, chatId)

		msgText := successText
		msg := TKeyboardMessageSend{
			ChatId:         chatId,
			Text:           msgText,
			KeyboardMarkup: GenerateMainKeyboard(),
		}
		env.users.saveLastUserAction(chatId, UserAction{
			Action:  lastAction,
			Context: nil,
		})
		return &msg, nil
	}
}

type timeekeepStoppedCallback func(chatId ChatId)

func (env *environment) onTimekeepStopped(chatId ChatId) {
	delete(env.timeKeepers, chatId)
	msgText := fmt.Sprintf("Focus session ended! You can take a break now")
	msg := TKeyboardMessageSend{
		ChatId:         chatId,
		Text:           msgText,
		KeyboardMarkup: GenerateMainKeyboard(),
	}

	env.marshalAndSendMessage(msg)
}

func (env *environment) marshalAndSendMessage(msg interface{}) {
	//Prepare message for sending
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	err = env.sendHttpMessage(msgBytes)
	if err != nil {
		log.Println(err)
		return
	}
}

func (env *environment) setupWebhook(certificateFilePath string, url string) error {
	keyFile, err := os.Open(certificateFilePath)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("certificate", keyFile.Name())
	io.Copy(part, keyFile)
	err = writer.WriteField("url", "https://"+url+"/")
	if err != nil {
		return err
	}
	err = writer.WriteField("ip_address", env.ipAddress)
	writer.Close()

	request, err := http.NewRequest("POST", "https://api.telegram.org/bot"+env.botKey+"/setWebhook", body)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	response, err := env.client.Do(request)
	if err != nil {
		return err
	}

	buf, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	log.Printf("reponse to the webhook instal - [%s]", buf)
	return nil
}

func (env *environment) deleteWebhook() error {
	resp, err := http.Get("https://api.telegram.org/bot" + env.botKey + "/deleteWebhook?url=https://" + env.ipAddress + "/")
	if err != nil {
		return err
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Printf("reponse to the webhook delete - [%s]", buf)
	return nil
}

func (env *environment) getWebhookInfo() error {
	resp, err := http.Get("https://api.telegram.org/bot" + env.botKey + "/getWebhookInfo?url=https://" + env.ipAddress + "/update")
	if err != nil {
		return err
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Printf("web hook info is - [%s]", buf)
	return nil
}

func (env *environment) sendHttpMessage(buf []byte) error {
	var resp *http.Response
	retryCounter := 0
	for {
		bufReader := bytes.NewReader(buf)
		request, err := http.NewRequest("POST", env.generateTelegramUrl("sendMessage"), bufReader)
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		resp, err = env.client.Do(request)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		} else if resp.StatusCode == http.StatusTooManyRequests && retryCounter < 5 {
			log.Printf("sent too many request, will sleep for 1 second. This is retry number %v", retryCounter)
			time.Sleep(time.Second)
			retryCounter++
			continue
		}

		desc, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send the message, error msg - [%v], status code - [%v], desciption - [%s]", err, resp.StatusCode, desc)
	}

	return nil
}

func createEnvironment(webhookAction string, botKey string, ipAddress string, certificateFilePath string, url string) *environment {
	//Valid input parameters
	if botKey == "" {
		log.Fatal("error: telegram bot token is not set")
	}
	if url == "" {
		log.Fatal("error: url is not set")
	}
	if ipAddress == "" {
		log.Fatal("error: ip address is not set")
	} else if !reIpAddress.MatchString(ipAddress) {
		log.Fatal("error: ip address is not valid")
	}

	env := environment{
		client:    http.Client{},
		botKey:    botKey,
		ipAddress: ipAddress,
		db:        &hDataBase{},
		users: Users{
			data: make(map[ChatId]User),
			mut:  sync.Mutex{},
		},
		timeKeepers: map[ChatId]*TimeKeeper{},
	}
	tmpString := ""
	env.db.initDB(&tmpString)
	var err error
	env.users.data, err = env.db.getAllUsersData()
	if err != nil {
		log.Fatal(err)
	}

	//process webhook action provided by the user
	if webhookAction == "install" {
		err := env.setupWebhook(certificateFilePath, url)
		if err != nil {
			log.Printf("error: failed to install webhook - %v", err)
		}
	} else if webhookAction == "delete" {
		err := env.deleteWebhook()
		if err != nil {
			log.Printf("error: failed to delete webhook - %v", err)
		}
	} else {
		err := env.getWebhookInfo()
		if err != nil {
			log.Printf("error: failed to get webhook info - %v", err)
		}
	}

	return &env
}

func GenerateMainKeyboard() TReplyKeyboard {
	keyboard := make([][]TKeyBoardButton, 3)
	keyboard[0] = GenerateKeyboardRow(TTEXT_START_FOCUS)
	keyboard[1] = GenerateKeyboardRow(TTEXT_START_BREAK)
	keyboard[2] = GenerateKeyboardRow(TTEXT_SETTINGS)

	return TReplyKeyboard{
		Keyboard:       keyboard,
		ResizeKeyboard: true,
	}
}

func GenerateCustomKeyboard(menuOptions ...string) TReplyKeyboard {
	keyboard := make([][]TKeyBoardButton, len(menuOptions))
	for i, option := range menuOptions {
		keyboard[i] = GenerateKeyboardRow(option)
	}

	return TReplyKeyboard{
		Keyboard:       keyboard,
		ResizeKeyboard: true,
	}
}

func GenerateKeyboardRow(btnText string) []TKeyBoardButton {
	keyboard := make([]TKeyBoardButton, 1)
	keyboard[0] = TKeyBoardButton{
		Text: btnText,
	}
	return keyboard
}

func GenerateListKeyboard(elems []string) TReplyKeyboard {
	fmt.Printf("DEBUG: GenerateListKeyboard: list %v\n", elems)

	keyboard := make([][]TKeyBoardButton, len(elems))
	for i, elem := range elems {
		keyboard[i] = GenerateKeyboardRow(elem)
	}
	return TReplyKeyboard{
		Keyboard:       keyboard,
		ResizeKeyboard: true,
	}
}

func (env *environment) generateTelegramUrl(action string) string {
	return "https://api.telegram.org/bot" + env.botKey + "/" + action
}
