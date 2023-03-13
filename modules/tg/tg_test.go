package tg

import (
	"log"
	"os"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"github.com/joho/godotenv"
)

func TestGetSummary(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}

	engine, err := database.Connect(os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASS"), os.Getenv("MYSQL_DB"))
	if err != nil {
		log.Fatal(err)
	}

	bot := new(Bot)
	err = bot.InitBot(os.Getenv("TELEGRAM_APITOKEN"), *engine)
	if err != nil {
		log.Fatal(err)
	}

	bot.TG_user = database.TgUser{
		L9Id: 316268749,
		TgId: 1086707888,
	}

	bot.GetSummary()
}
