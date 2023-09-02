package main

import (
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/notify"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
)

func main() {
	ssau_parser.HeadURL = "https://ssau.ru"
	if err := tg.CheckEnv(); err != nil {
		log.Fatal(err)
	}
	logs := database.OpenLogs()
	defer logs.CloseAll()
	log.SetOutput(io.MultiWriter(os.Stderr, logs.ErrorFile))
	help, err := os.ReadFile("help.txt")
	if err != nil {
		log.Fatal(err)
	}

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
	bot.TestUser, err = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	bot.WkPath = os.Getenv("WK_PATH")
	bot.HelpTxt = string(help)
	//now, _ := time.Parse("2006-01-02 15:04 -07", "2023-02-07 07:00 +04")
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), (now.Minute() + 1), 0, 0, now.Location())
	// TODO: что-то придумать с этим выжиданием
	log.Println("Waiting...")
	time.Sleep(next.Sub(now))
	mailTicker := time.NewTicker(1 * time.Minute)
	sheduleTicker := time.NewTicker(30 * time.Minute)
	log.Println("Started")
	defer mailTicker.Stop()
	defer sheduleTicker.Stop()
	for {
		select {
		case update := <-*bot.Updates:
			now = time.Now()
			_, err := bot.HandleUpdate(update, now)
			if err != nil {
				log.Println(err)
			}
		case <-mailTicker.C:
			now = time.Now()
			log.Println(now)
			notes, err := notify.CheckNext(bot.DB, now)
			if err != nil {
				log.Println(err)
			}
			notify.FirstMailing(bot, now)
			notify.Mailing(bot, notes, now)
			notify.ClearTemp(bot, now)
		case <-sheduleTicker.C:
			now = time.Now()
			if now.Hour() > 8 && now.Hour() < 20 {
				go notify.CheckShedules(bot, now)
			}
		}
	}
}
