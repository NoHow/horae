package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Config struct {
	TelegramBotToken string `json:"telegram-bot-token"`
	CertificateFile  string `json:"certificate-file"`
	KeyFile          string `json:"key-file"`
	Url              string `json:"url"`
	IpAddress        string `json:"ip-address"`
}

func loadConfig() Config {
	cfgFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	cfg := Config{}
	err = json.Unmarshal(cfgFile, &cfg)
	if err != nil {
		log.Fatalf("error: failed to parse config %v", err)
	}

	return cfg
}

func main() {
	tlsCert := os.Getenv("tls-certificate")
	fmt.Println(tlsCert)

	webHookAction := flag.String("webhook", "", "install or delete webhook, empty string means no action")
	flag.Parse()

	cfg := loadConfig()

	env := createEnvironment(*webHookAction, cfg.TelegramBotToken, cfg.IpAddress, cfg.CertificateFile, cfg.Url)
	if env == nil {
		log.Fatal("error: failed to create environment")
	}
	http.HandleFunc("/update/", env.updateHandler)
	http.HandleFunc("/", env.rootHandler)

	log.Fatal(http.ListenAndServeTLS(":443", cfg.CertificateFile, cfg.KeyFile, nil))
}