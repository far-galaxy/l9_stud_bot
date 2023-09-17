package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/notify"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/parser"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
	"github.com/robfig/cron/v3"
)

var build string
var mainbot *tg.Bot

func main() {
	if err := tg.CheckEnv(); err != nil {
		log.Fatal(err)
	}
	parser.HeadURL = os.Getenv("RASP_URL")
	logs := database.OpenLogs()
	defer logs.CloseAll()
	log.SetOutput(io.MultiWriter(os.Stderr, logs.ErrorFile))
	help, err := os.ReadFile("help.txt")
	if err != nil {
		log.Fatal(err)
	}

	// bot.Debug = log.New(io.MultiWriter(os.Stderr, database.CreateLog("messages")), "", log.LstdFlags)
	mainbot, err = tg.InitBot(
		logs,
		database.DB{
			User:   os.Getenv("MYSQL_USER"),
			Pass:   os.Getenv("MYSQL_PASS"),
			Schema: os.Getenv("MYSQL_DB"),
		},
		os.Getenv("TELEGRAM_APITOKEN"),
		build,
	)
	if err != nil {
		log.Fatal(err)
	}
	mainbot.Week, err = strconv.Atoi(os.Getenv("START_WEEK"))
	if err != nil {
		log.Fatal(err)
	}
	mainbot.TestUser, err = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	mainbot.WkPath = os.Getenv("WK_PATH")
	mainbot.HelpTxt = string(help)
	c := cron.New()
	_, err = c.AddFunc("0/5 6-22 * * *", notifications)
	if err != nil {
		log.Fatal(err)
	}
	shedulePeriod, err := strconv.Atoi(os.Getenv("SHEDULES_CHECK_PERIOD"))
	if err != nil {
		log.Fatal(err)
	}
	_, err = c.AddFunc(fmt.Sprintf("@every %dm", shedulePeriod), sheduleCheck)
	if err != nil {
		log.Fatal(err)
	}
	c.Start()
	sheduleTicker := time.NewTicker(time.Duration(shedulePeriod) * time.Minute)
	log.Println("Started")
	defer sheduleTicker.Stop()
	for update := range *mainbot.Updates {
		now := time.Now()
		_, err := mainbot.HandleUpdate(update, now)
		if err != nil {
			log.Println(err)
		}
	}
}

func notifications() {
	now := time.Now()
	//now := time.Date(2023, 9, 15, 17, 20, 0, 0, time.Local)
	log.Println(now)
	notes, err := notify.CheckNext(mainbot.DB, now)
	if err != nil {
		log.Println(err)
	}
	notify.Mailing(mainbot, notes, now)
	notify.FirstMailing(mainbot, now)
	notify.ClearTemp(mainbot, now)
}

func sheduleCheck() {
	now := time.Now()

	if now.Hour() > 8 && now.Hour() < 20 {
		notify.CheckShedules(mainbot, now)
	}
}
