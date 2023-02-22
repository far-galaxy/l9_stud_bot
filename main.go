package main

import (
	"log"
	"os"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}

	engine := database.Connect(os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASS"), os.Getenv("MYSQL_DB"))

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			var users []database.TgUser
			err := engine.Find(&users, &database.TgUser{TgId: update.Message.Chat.ID})
			if err != nil {
				log.Fatal(err)
			}

			var tg_user database.TgUser
			if len(users) == 0 {
				l9id := database.GenerateID(engine)

				user := database.User{
					L9Id: l9id,
				}

				tg_user = database.TgUser{
					L9Id:   l9id,
					Name:   update.Message.From.UserName,
					TgId:   update.Message.Chat.ID,
					PosTag: "not_started",
				}
				_, err := engine.Insert(user, tg_user)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				tg_user = users[0]
			}

			if tg_user.PosTag == "not_started" {
				tg_user.PosTag = "started"
				_, err := engine.Update(tg_user)
				if err != nil {
					log.Fatal(err)
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hello!")
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Oops!")
				bot.Send(msg)
			}

		}
	}
}
