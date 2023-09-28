package tg

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
)

var TestDB = database.DB{
	User:   "root",
	Pass:   "18064",
	Schema: "l9_db_test",
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
	envKeys = append(envKeys, "LOST_KEY")
	if err := CheckEnv(); err != nil {
		log.Println(err)
		envKeys = envKeys[:len(envKeys)-1]
	}
	t.Log("ok")
}

func InitTestBot(files database.LogFiles) *Bot {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}
	bot, err := InitBot(files, TestDB, os.Getenv("TELEGRAM_APITOKEN"), "test")
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
	InitTestBot(files)

	// Тестируем неправильный токен
	_, err := InitBot(files, TestDB, os.Getenv("TELEGRAM_APITOKEN")+"oops", "test")
	if err != nil {
		log.Println(err)
	}
	t.Log("ok")
}

func TestInitUser(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := InitTestBot(files)

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
	t.Log("ok")
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
	bot := InitTestBot(files)

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
	ssauparser.HeadURL = "http://127.0.0.1:5000/prod"
	// Бот общается с ботом
	for i, query := range dialog {
		if i == len(dialog)-1 {
			ssauparser.HeadURL = "https://sasau.ru"
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
	t.Log("ok")
}

var times = []string{
	"2023-02-06 11:40 +04",
	"2023-02-06 11:40 +04",
	"2023-02-06 13:10 +04",
	"2023-02-06 13:35 +04",
	"2023-02-06 15:20 +04",
	"2023-02-06 16:55 +04",
	"2023-02-07 16:55 +04",
}

func TestSummary(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := InitTestBot(files)

	user := TestUser
	user.ID, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &user,
		},
	}
	ssauparser.HeadURL = "http://127.0.0.1:5000/prod"
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
	t.Log("ok")
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

func TestGetWeekLessons(t *testing.T) {
	ssauparser.HeadURL = "http://127.0.0.1:5000/prod"
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := InitTestBot(files)
	bot.Week = 5
	bot.WkPath = "C:\\Program Files\\wkhtmltopdf\\bin\\wkhtmltoimage.exe"
	user := database.TgUser{}
	user.TgId, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	shedules := []ssauparser.WeekShedule{
		{
			SheduleID: 100000000,
			IsGroup:   true,
			Week:      1,
		},
		{
			SheduleID: 3,
			IsGroup:   false,
			Week:      1,
		},
	}
	now, _ := time.Parse("2006-01-02 15:04 -07", times[2])
	for _, sh := range shedules {
		err := sh.DownloadByID(true)
		if err != nil {
			log.Fatal(err)
		}
		_, _, err = ssauparser.UpdateSchedule(bot.DB, sh)
		if err != nil {
			log.Fatal(err)
		}
		err = bot.CreateWeekImg(now, &user, Swap(sh), 0, false, "")
		if err != nil {
			log.Fatal(err)
		}
	}
	t.Log("ok")
}
