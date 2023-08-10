package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
)

func main() {
	ssau_parser.HeadURL = "http://127.0.0.1:5000/prod"
	if err := tg.CheckEnv(); err != nil {
		log.Fatal(err)
	}
	logs := database.OpenLogs()
	defer logs.CloseAll()
	//bot := new(tg.Bot)
	// bot.Debug = log.New(io.MultiWriter(os.Stderr, database.CreateLog("messages")), "", log.LstdFlags)
	bot, err := tg.InitBot(
		logs,
		database.DB{
			User:   os.Getenv("MYSQL_USER"),
			Pass:   os.Getenv("MYSQL_PASS"),
			Schema: os.Getenv("MYSQL_DB"),
		},
		os.Getenv("TELEGRAM_APITOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	bot.Week, err = strconv.Atoi(os.Getenv("START_WEEK"))
	if err != nil {
		log.Fatal(err)
	}
	bot.WkPath = os.Getenv("WK_PATH")
	now, _ := time.Parse("2006-01-02 15:04 -07", "2023-02-06 11:20 +04")
	for update := range *bot.Updates {
		_, err := bot.HandleUpdate(update, now)
		if err != nil {
			log.Println(err)
		}
	}
}
