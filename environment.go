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
	tmpTasks    map[ChatId]Task
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
	ParseMode      string         `json:"parse_mode"`
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
	TTEXT_START_COMMAND     = "/start"
	TTEXT_DURATIONS_COMMAND = "/durations"
	TTEXT_MAIN_MENU_COMMAND = "/main"

	TTEXT_MAIN_MENU             = "Main menu"
	TTEXT_START_FOCUS           = "Let's focus"
	TTEXT_SETTINGS              = "Settings"
	TTEXT_STOP_FOCUS            = "Stop focus"
	TTEXT_TIME_LEFT             = "Time left"
	TTEXT_START_BREAK           = "Let's take a break"
	TTEXT_STOP_BREAK            = "Stop break"
	TTEXT_FOCUS_DURATION        = "Focus duration"
	TTEXT_BREAK_DURATION        = "Break duration"
	TTEXT_CHANGE_FOCUS_DURATION = "Change focus duration"
	TTEXT_CHANGE_BREAK_DURATION = "Change break duration"
	TTEXT_CHANGE_WORKDAY        = "Workday"
	TTEXT_ADD_TASK              = "Add task"
	TTEXT_EDIT_TASK_NAME        = "Edit task name"
	TTEXT_EDIT_TASK_PERIODS     = "Edit the amount of periods"
	TTEXT_EDIT_TASK_ORDER       = "Edit task order"
	TTEXT_DELETE_TASK           = "Delete task"
	TTEXT_BACK                  = "Back"
)

const (
	INVALID_ACTION = iota
	START_ACTION
	CHANGE_FOCUS_DURATION_ACTION
	CHANGE_BREAK_DURATION_ACTION
	SELECT_TASK_NAME_ACTION
	SELECT_TASK_PERIODS_ACTION
	SELECT_TASK_ORDER_ACTION
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
	focusDurations := []string{"15 minutes", "30 minutes", "45 minutes", "1 hour"}
	pauseDurations := []string{"5 minutes", "10 minutes", "15 minutes", "20 minutes"}
	var processedResult MenuProcessorResult
	switch Update.Message.Text {
	case TTEXT_START_COMMAND:
		newUser := User{
			FirstName: Update.Message.From.FirstName,
		}
		isNewUser := env.users.add(Update.GetChatId(), newUser)
		if isNewUser {
			processedResult.responseType = RESPONSE_TYPE_KEYBOARD
			processedResult.replyText = fmt.Sprintf("Hello %s! I will help you to keep organised with your time!\n"+
				"Please select how long you want your focus duration to be?", Update.Message.From.FirstName)
			processedResult.replyKeyboard = GenerateCustomKeyboard(focusDurations...)
			processedResult.userAction = UserAction{CurrentMenu: MENU_MAIN_MENU}
		}
	case TTEXT_MAIN_MENU_COMMAND:
		fmt.Printf("User %v selected main menu\n", Update.GetChatId())
		_, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}
		processedResult.responseType = RESPONSE_TYPE_KEYBOARD
		processedResult.replyText = "Main menu"
		processedResult.replyKeyboard = GenerateMainKeyboard()
		processedResult.userAction = UserAction{CurrentMenu: MENU_MAIN_MENU}
	case TTEXT_DURATIONS_COMMAND:
		user, ok := env.users.data[Update.GetChatId()]
		if !ok {
			log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
			return
		}
		processedResult.responseType = RESPONSE_TYPE_KEYBOARD
		processedResult.replyText = fmt.Sprintf("Your focus duration is %v and your break duration is %v minutes", user.FocusDurationMins, user.BreakDurationMins)
		processedResult.replyKeyboard = GenerateMainKeyboard()
		processedResult.userAction = UserAction{CurrentMenu: MENU_MAIN_MENU}
	}

	if processedResult.responseType == RESPONSE_TYPE_NONE {
		user, ok := env.users.data[Update.GetChatId()]
		if !ok {
			msgText := fmt.Sprintf("Oops! I don't know you yet. Please type %v to start", TTEXT_START_COMMAND)
			Msg = TMessageSend{
				ChatId: Update.GetChatId(),
				Text:   msgText,
			}
		} else {
			switch user.LastAction.CurrentMenu {
			case MENU_MAIN_MENU:
				processedResult, err = processMainMenu(Update.Message.Text, user, Update.GetChatId(), env)
			case MENU_INFOCUS:
				processedResult, err = processInFocusMenu(Update.Message.Text, Update.GetChatId(), &env.timeKeepers)
			case MENU_INBREAK:
				processedResult, err = processInBreakMenu(Update.Message.Text, Update.GetChatId(), &env.timeKeepers)
			case MENU_INIT_FOCUS:
				processedResult, err = processInitFocusMenu(Update.Message.Text, Update.GetChatId(), user, &env.users, focusDurations, pauseDurations)
			case MENU_INIT_BREAK:
				processedResult, err = processInitBreakMenu(Update.Message.Text, Update.GetChatId(), user, &env.users, pauseDurations)
			case MENU_SETTINGS:
				processedResult, err = processSettingsMenu(Update.Message.Text, user)
			case MENU_SETTINGS_FOCUS_DURATION:
				processedResult, err = processSettingsFocusDurationMenu(Update.Message.Text, Update.GetChatId(), user, &env.users, focusDurations)
			case MENU_SETTINGS_BREAK_DURATION:
				processedResult, err = processSettingsBreakDurationMenu(Update.Message.Text, Update.GetChatId(), user, &env.users, pauseDurations)
			case MENU_SETTINGS_WORKDAY:
				processedResult, err = processSettingsWorkdayMenu(Update.Message.Text, Update.GetChatId(), user, &env.users)
			case MENU_SETTINGS_WORKDAY_TASK_EDIT:
				processedResult, err = processSettingsWorkdayTaskEditMenu(Update.Message.Text, Update.GetChatId(), user, &env.users)
			}

			if err != nil {
				log.Println(err)
				return
			}
		}
	}

	user, ok := env.users.data[Update.GetChatId()]
	if !ok {
		log.Printf("user with chat id - [%v] is not found", Update.GetChatId())
	} else {
		env.users.saveLastUserAction(Update.GetChatId(), processedResult.userAction)
		env.db.saveUserData(Update.GetChatId(), user)
	}

	fmt.Printf("Processed result - [%v]", processedResult)
	switch processedResult.responseType {
	case RESPONSE_TYPE_KEYBOARD:
		keyboardMsg = TKeyboardMessageSend{
			ChatId:         Update.GetChatId(),
			Text:           processedResult.replyText,
			KeyboardMarkup: processedResult.replyKeyboard,
			ParseMode:      "HTML",
		}
		env.marshalAndSendMessage(keyboardMsg)
	case RESPONSE_TYPE_TEXT:
		Msg = TMessageSend{
			ChatId: Update.GetChatId(),
			Text:   processedResult.replyText,
		}
		env.marshalAndSendMessage(Msg)
	}
	log.Printf("Successfully processed message from user - [%v]", Update.Message.From.FirstName)
}

type timeekeepStoppedCallback func(chatId ChatId, finishMessage string)

func (env *environment) onTimekeepStopped(chatId ChatId, finishMessage string) {
	delete(env.timeKeepers, chatId)
	msg := TKeyboardMessageSend{
		ChatId:         chatId,
		Text:           finishMessage,
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
		tmpTasks:    map[ChatId]Task{},
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
		Keyboard:        keyboard,
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}
}

func generateWorkdaySettingsKeyboard(keyboardFields []string) TReplyKeyboard {
	keyboardFields = append(keyboardFields, TTEXT_ADD_TASK)
	keyboardFields = append(keyboardFields, TTEXT_MAIN_MENU)
	return GenerateCustomKeyboard(keyboardFields...)
}

func GenerateKeyboardRow(btnText string) []TKeyBoardButton {
	keyboard := make([]TKeyBoardButton, 1)
	keyboard[0] = TKeyBoardButton{
		Text: btnText,
	}
	return keyboard
}

func (env *environment) generateTelegramUrl(action string) string {
	return "https://api.telegram.org/bot" + env.botKey + "/" + action
}
