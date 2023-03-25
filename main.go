package main

import (
	"log"
	"os"
	"strconv"
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
	defer engine.Close()

	bot := new(tg.Bot)
	bot.Week = 5
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
			} else if msg.Text == "Главное меню" {
				bot.GetPersonalSummary()
			} else if tg_user.PosTag == "add" || tg_user.PosTag == "ready" {
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

			if query.Data == "cancel" {
				bot.Cancel(query)
			} else if strings.Contains(tg_user.PosTag, "confirm_add") {
				bot.Confirm(query)
			} else if strings.Contains(tg_user.PosTag, "confirm_see") {
				bot.SeeShedule(query)
				bot.DeleteMsg(query)
			} else if strings.Contains(query.Data, "day") {
				// TODO: Зарефакторить
				data := strings.Split(query.Data, "_")
				if data[1] == "personal" {
					dt, err := strconv.ParseInt(data[2], 0, 64)
					if err != nil {
						log.Fatal(err)
					}
					bot.GetPersonalDaySummary(int(dt), *query.Message)
				} else {
					shedule, dt, err := tg.ParseQuery(data)
					if err != nil {
						log.Fatal(err)
					}
					bot.GetDaySummary(shedule, dt, false, *query.Message)
				}
			} else if strings.Contains(query.Data, "near") {
				data := strings.Split(query.Data, "_")
				if data[1] == "personal" {
					bot.GetPersonalSummary(*query.Message)
				} else {
					shedule, _, err := tg.ParseQuery(data)
					if err != nil {
						log.Fatal(err)
					}
					bot.GetSummary(shedule, false, *query.Message)
				}
			} else if strings.Contains(query.Data, "week") {
				data := strings.Split(query.Data, "_")
				shedule, dw, err := tg.ParseQuery(data)
				if err != nil {
					log.Fatal(err)
				}
				bot.GetWeekSummary(shedule, dw, false, *query.Message)
			}
		}
	}
}
