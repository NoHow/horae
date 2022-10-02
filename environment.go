package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
)

var validPath = regexp.MustCompile("^/(update)/+")
var reIpAddress = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`)

type environment struct {
	client http.Client
	botKey string
	ipAddress string
}

func getPathValue(r *http.Request, pathCheck *regexp.Regexp) (string, error) {
	m := pathCheck.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return "", fmt.Errorf("url path is not valid")
	}

	log.Printf("getPathValue will return %v", m[1])
	return m[1], nil
}

func (env *environment) updateHandler(w http.ResponseWriter, r *http.Request) {
	pageTitle, err := getPathValue(r, validPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Received page title - [%v]", pageTitle)
}

func (env *environment) rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("I'm alive!")
}

func (env *environment) setupWebhook(certificateFilePath string) error {
	keyFile, err := os.Open(certificateFilePath)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("certificate", keyFile.Name())
	io.Copy(part, keyFile)
	err = writer.WriteField("url", "https://" + env.ipAddress + "/")
	if err != nil {
		return err
	}
	err = writer.WriteField("ip_address", env.ipAddress)
	writer.Close()

	request, err := http.NewRequest("POST", "https://api.telegram.org/bot" + env.botKey + "/setWebhook", body)
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
	log.Printf("reponse to the webhook instal - [%s]", buf)
	return nil
}

func (env *environment) getWebhookInfo() error {
	resp, err := http.Get("https://api.telegram.org/bot" + env.botKey + "/getWebhookInfo?url=https://" + env.ipAddress +  "/update")
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

func createEnvironment(webhookAction string, botKey string, ipAddress string, certificateFilePath string) *environment {
	//Valid input parameters
	if botKey == "" {
		log.Fatal("error: telegram bot token is not set")
	}
	if ipAddress == "" {
		log.Fatal("error: ip address is not set")
	} else if !reIpAddress.MatchString(ipAddress) {
		log.Fatal("error: ip address is not valid")
	}

	env := environment{
		client: http.Client{},
		botKey: botKey,
		ipAddress: ipAddress,
	}

	//process webhook action provided by the user
	if webhookAction == "install" {
		err := env.setupWebhook(certificateFilePath)
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