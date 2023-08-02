package main

import (
	"log"
	"os"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	tg.CheckEnv()

	engine, err := database.Connect(os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASS"), os.Getenv("MYSQL_DB"))
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	//bot := new(tg.Bot)
	// bot.Week = 5
	// bot.WkPath = os.Getenv("WK_PATH")
	// bot.Debug = log.New(io.MultiWriter(os.Stderr, database.CreateLog("messages")), "", log.LstdFlags)
	bot, err := tg.InitBot(
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASS"),
		os.Getenv("MYSQL_DB"),
		os.Getenv("TELEGRAM_APITOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	bot.GetUpdates()
	for update := range *bot.Updates {
		handle(bot, update)
	}
	/*
		for update := range *bot.Updates {
			if update.Message != nil {
				msg := update.Message
				bot.Debug.Printf("Message [%s] %s", msg.From.UserName, msg.Text)

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
				bot.Debug.Printf("Callback [%s] %s", query.From.UserName, query.Data)

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
				} else if tg.KeywordContains(query.Data, tg.SumKey) {
					bot.HandleSummary(query)
				}
			}
		}*/
}

func handle(bot *tg.Bot, update tgbotapi.Update) {
	if update.Message != nil {
		msg := update.Message
		bot.Debug.Printf("Message [%s] %s", msg.From.UserName, msg.Text)
	}
}
