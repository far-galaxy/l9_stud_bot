package main

import (
	"log"
	"os"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}

	engine, err := database.Connect(os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASS"), os.Getenv("MYSQL_DB"))
	if err != nil {
		log.Fatal(err)
	}

	bot := new(tg.Bot)
	err = bot.InitBot(os.Getenv("TELEGRAM_APITOKEN"), *engine)
	if err != nil {
		log.Fatal(err)
	}

	updates := bot.GetUpdates()

	for update := range *updates {
		if update.Message != nil {
			msg := update.Message
			log.Printf("Message [%s] %s", msg.From.UserName, msg.Text)

			tg_user, err := bot.InitUser(msg.From.ID, msg.From.UserName)
			if err != nil {
				log.Fatal(err)
			}

			if tg_user.PosTag == "not_started" {
				bot.Start()
			} else if tg_user.PosTag == "add" {
				bot.Find(msg.Text)
			} else {
				bot.Etc()
			}

		}

		if update.CallbackQuery != nil {
			query := update.CallbackQuery
			log.Printf("Callback [%s] %s", query.From.UserName, query.Data)

			tg_user, err := bot.InitUser(query.From.ID, query.From.UserName)
			if err != nil {
				log.Fatal(err)
			}

			if strings.Contains(tg_user.PosTag, "confirm") {
				bot.Confirm(query, tg_user, tg_user.PosTag == "confirm_group")
			}
		}
	}
}
