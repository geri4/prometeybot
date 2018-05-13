package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/asdine/storm"
	simplejson "github.com/bitly/go-simplejson"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	// APIEndpoint is the endpoint for all API methods,
	// with formatting for Sprintf.
	APIEndpoint = "https://api.telegram.org/bot%s/%s"
	// FileEndpoint is the endpoint for downloading a file from Telegram.
	FileEndpoint = "https://api.telegram.org/file/bot%s/%s"
)

// Chat Struct for chats
type Chat struct {
	ID        int64
	Auth      bool
	FirstName string
	LastName  string
}

// Alert struct for alerts
type Alert struct {
	status   string
	name     string
	host     string
	severity string
	env      string
}

func main() {
	var apikey = flag.String("apikey", "", "Specify telegram bot api key")
	var chatPassword = flag.String("chatpassword", "", "Specify telegram chat password for user authentication")
	var dbPath = flag.String("dbpath", "/data/chat.db", "Specify telegram bot database path")
	flag.Parse()
	if os.Getenv("APIKEY") != "" {
		*apikey = os.Getenv("APIKEY")
	}
	if os.Getenv("CHATPASSWORD") != "" {
		*chatPassword = os.Getenv("CHATPASSWORD")
	}
	if os.Getenv("DBPATH") != "" {
		*dbPath = os.Getenv("DBPATH")
	}
	bot, err := tgbotapi.NewBotAPI(*apikey)
	if err != nil {
		log.Panic(err)
	}
	db, err := storm.Open(*dbPath)
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
	go telegram(db, bot, *chatPassword)
	http.HandleFunc("/", sendalert(db, bot))
	err = http.ListenAndServe(":9010", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func telegram(db *storm.DB, bot *tgbotapi.BotAPI, chatPassword string) {

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, _ := bot.GetUpdatesChan(u)

	var chat Chat
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		err := db.One("ID", update.Message.Chat.ID, &chat)
		if err != nil {
			chat = Chat{
				ID:        update.Message.Chat.ID,
				Auth:      false,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
			}
			db.Save(&chat)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi! Please enter the password:")
			bot.Send(msg)
			continue
		}

		if chat.Auth == false && update.Message.Text == chatPassword {
			err = db.UpdateField(&Chat{ID: update.Message.Chat.ID}, "Auth", true)
			if err != nil {
				log.Panic(err)
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Passsord is valid. Welcome!")
			bot.Send(msg)
			continue
		} else if chat.Auth == false && update.Message.Text != chatPassword {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Passsord is invalid. Please enter valid password!")
			bot.Send(msg)
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		bot.Send(msg)
	}
}

func sendalert(db *storm.DB, bot *tgbotapi.BotAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var Chats []Chat
		err := db.Find("Auth", true, &Chats)
		if err != nil {
			log.Panic(err)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(body))
		js, err := simplejson.NewJson(body)
		if err != nil {
			fmt.Println(err)
		}

		// parse alerts from json sent by alertmanager
		alertlen := len(js.Get("alerts").MustArray())
		var alert Alert
		var message string
		for i := 0; i < alertlen; i++ {
			alert.status = js.Get("alerts").GetIndex(i).Get("status").MustString()
			alert.name = js.Get("alerts").GetIndex(i).Get("annotations").Get("summary").MustString()
			alert.host = js.Get("alerts").GetIndex(i).Get("labels").Get("container_name").MustString()
			if alert.host == "" {
				alert.host = js.Get("alerts").GetIndex(i).Get("labels").Get("node").MustString()
			}
			if alert.host == "" {
				alert.host = js.Get("alerts").GetIndex(i).Get("labels").Get("host").MustString()
			}
			alert.severity = js.Get("alerts").GetIndex(i).Get("labels").Get("severity").MustString()
			alert.env = js.Get("alerts").GetIndex(i).Get("labels").Get("environment").MustString()
			message = message + "\n\n" + strings.ToUpper(alert.status) + ": " + alert.name + "\nHostname: " +
				alert.host + "\nSeverity: " + alert.severity
		}

		// send alerts for all users subscribed to bot
		for _, chat := range Chats {
			fmt.Println(chat.ID)
			msg := tgbotapi.NewMessage(chat.ID, message)
			bot.Send(msg)
		}
	}
}
