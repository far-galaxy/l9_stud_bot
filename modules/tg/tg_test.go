package tg

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var TestDB = database.DB{
	User:   "test",
	Pass:   "TESTpass1!",
	Schema: "testdb",
}

var TestUser = tgbotapi.User{
	ID:        12345,
	FirstName: "Grzegorz",
	LastName:  "Brzbrz",
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

func initTestBot(files database.LogFiles) *Bot {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}
	bot, err := InitBot(files, TestDB, os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("l9id >= 0").Delete(&database.TgUser{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("teacherid >= 0").Delete(&database.Teacher{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("groupid >= 0").Delete(&database.Group{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("lessonid >= 0").Delete(&database.Lesson{})
	if err != nil {
		log.Fatal(err)
	}
	return bot
}
func TestInitBot(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	initTestBot(files)

	// Тестируем неправильный токен
	_, err := InitBot(files, TestDB, os.Getenv("TELEGRAM_APITOKEN")+"oops")
	if err != nil {
		log.Println(err)
	}
}

func TestInitUser(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := initTestBot(files)

	// Я новенький
	_, err := InitUser(bot.DB, &TestUser)
	if err != nil {
		log.Fatal(err)
	}
	// Я уже Смешарик
	_, err = InitUser(bot.DB, &TestUser)
	if err != nil {
		log.Fatal(err)
	}
}

var dialog = []string{
	"/start",
	"1000",
	"Павлов",
	"100",
	"Иванов",
	"aaa",
	"aaa",
}

func TestHandleUpdate(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := initTestBot(files)

	user := TestUser
	user.ID, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	now, _ := time.Parse("2006-01-02 15:04 -07", "2023-03-06 11:20 +04")
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &user,
			Date: int(now.Unix()),
		},
	}
	var messages []tgbotapi.Message
	ssau_parser.HeadURL = "http://127.0.0.1:5000/prod"
	// Бот общается с ботом
	for i, query := range dialog {
		if i == len(dialog)-1 {
			ssau_parser.HeadURL = "https://sasau.ru"
		}
		update.Message.Text = query
		msg, err := bot.HandleUpdate(update)
		if err != nil {
			log.Fatal(err)
		}
		messages = append(messages, msg)
	}

	// Бот нажимает на кнопки за пользователя
	update = tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			From:    &user,
			Message: &messages[3],
			Data:    *messages[3].ReplyMarkup.InlineKeyboard[0][0].CallbackData,
		},
	}
	_, err := bot.HandleUpdate(update, now)
	if err != nil {
		log.Println(err)
	}

	// Галя, отмена!
	update = tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			From:    &user,
			Message: &messages[4],
			Data:    "cancel",
		},
	}
	_, err = bot.HandleUpdate(update, now)
	if err != nil {
		log.Println(err)
	}
}

var times = []string{
	"2023-03-06 11:40 +04",
	"2023-03-06 11:40 +04",
	"2023-03-06 13:10 +04",
	"2023-03-06 13:35 +04",
	"2023-03-06 15:20 +04",
	"2023-03-06 16:55 +04",
	"2023-03-07 16:55 +04",
}

func TestSummary(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := initTestBot(files)

	user := TestUser
	user.ID, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &user,
		},
	}
	ssau_parser.HeadURL = "http://127.0.0.1:5000/prod"
	// Ещё немного общения в разное время
	var messages []tgbotapi.Message
	for _, te := range times {
		now, _ := time.Parse("2006-01-02 15:04 -07", te)
		update.Message.Text = dialog[1]
		update.Message.Date = int(now.Unix())
		msg, err := bot.HandleUpdate(update)
		if err != nil {
			log.Fatal(err)
		}
		messages = append(messages, msg)
	}

	// Обновляем карточку за пользователя
	now, _ := time.Parse("2006-01-02 15:04 -07", times[2])
	update = tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			From:    &user,
			Message: &messages[1],
			Data:    *messages[1].ReplyMarkup.InlineKeyboard[0][0].CallbackData,
		},
	}
	_, err := bot.HandleUpdate(update, now)
	if err != nil {
		log.Println(err)
	}
	/* Оставим это на всякий случай
	log.Println("Нажми на кнопку - получишь результат!")
	bot.GetUpdates()
	upd := <-*bot.Updates
	now, _ = time.Parse("2006-01-02 15:04 -07", "2023-03-07 16:55 +04")
	_, err = bot.HandleUpdate(upd, now)
	if err != nil {
		log.Fatal(err)
	}*/
}
