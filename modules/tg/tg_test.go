package tg

import (
	"log"
	"os"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var TestDB = database.DB{
	User:   "test",
	Pass:   "TESTpass1!",
	Schema: "testdb",
}

func TestCheckEnv(t *testing.T) {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}

	// Добавляем несуществующий ключ
	env_keys = append(env_keys, "LOST_KEY")
	if err := CheckEnv(); err != nil {
		log.Println(err)
		env_keys = env_keys[:len(env_keys)-1]
	}
}

func initTestBot() *Bot {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}
	bot, err := InitBot(TestDB, os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("l9id >= 0").Delete(&database.TgUser{})
	if err != nil {
		log.Fatal(err)
	}
	return bot
}
func TestInitBot(t *testing.T) {
	initTestBot()

	// Тестируем неправильный токен
	_, err := InitBot(TestDB, os.Getenv("TELEGRAM_APITOKEN")+"oops")
	if err != nil {
		log.Println(err)
	}
}

func TestInitUser(t *testing.T) {
	bot := initTestBot()

	// Я новенький
	user := tgbotapi.User{
		ID:        12345,
		FirstName: "Grzegorz",
		LastName:  "Brzbrz",
	}
	_, err := InitUser(bot.DB, &user)
	if err != nil {
		log.Fatal(err)
	}
	// Я уже Смешарик
	_, err = InitUser(bot.DB, &user)
	if err != nil {
		log.Fatal(err)
	}
}

func TestHandleUpdate(t *testing.T) {
	bot := initTestBot()

	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &tgbotapi.User{
				ID:        12345,
				FirstName: "Grzegorz",
				LastName:  "Brzbrz",
			},
			Text: "start",
		},
	}
	err := bot.HandleUpdate(update)
	if err != nil {
		log.Fatal(err)
	}
}
