package main

import (
	"log"
	"os"

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
	bot.InitBot(os.Getenv("TELEGRAM_APITOKEN"), *engine)

	updates := bot.GetUpdates()

	for update := range *updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			tg_user, err := bot.InitUser(update.Message)
			if err != nil {
				log.Fatal(err)
			}

			if tg_user.PosTag == "not_started" {
				bot.Start()
			} else if tg_user.PosTag == "add" {
				bot.Find(update.Message.Text)
			} else {
				bot.Etc()
			}

		}
	}
}
