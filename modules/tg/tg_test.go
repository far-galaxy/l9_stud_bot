package tg

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/htmlschedule"
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

func InitTestBot() *Bot {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}
	DB := database.DB{
		User:   os.Getenv("MYSQL_USER"),
		Pass:   os.Getenv("MYSQL_PASS"),
		Schema: os.Getenv("MYSQL_DB"),
	}
	bot, err := InitBot(DB, os.Getenv("TELEGRAM_APITOKEN"), "test")
	bot.StartTxt = "Привет!"
	bot.HelpTxt = "Ну тут наши полномочия всё..."
	bot.Week = 5
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("l9id >= 0").Delete(&database.TgUser{})
	if err != nil {
		log.Fatal(err)
	}
	_, err = bot.DB.Where("teacherid >= 0").Delete(&database.Staff{})
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
	InitTestBot()

	// Тестируем неправильный токен
	_, err := InitBot(TestDB, os.Getenv("TELEGRAM_APITOKEN")+"oops", "test")
	if err != nil {
		log.Println(err)
	}
	t.Log("ok")
}

func TestInitUser(t *testing.T) {
	bot := InitTestBot()

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
	"1001",
	"Иванов",
	"aaa",
	"100",
	"/start",
	"/help",
	"/keyboard",
	"/session",
	"Настройки",
	"aaa",
}

const TestServer = "http://127.0.0.1:5000/prod"

func TestHandleUpdate(t *testing.T) {
	bot := InitTestBot()

	user := TestUser
	user.ID, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	now, _ := time.Parse("2006-01-02 15:04 -07", "2023-02-06 11:20 +04")
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &user,
			Date: int(now.Unix()),
		},
	}
	var messages []tgbotapi.Message
	ssauparser.HeadURL = TestServer
	// Бот общается с ботом
	for i, query := range dialog {
		if i == len(dialog)-1 {
			ssauparser.HeadURL = "https://sasau.ru"
		}
		update.Message.Text = query
		update.Message.Chat = &tgbotapi.Chat{Type: "private"}
		msg, err := bot.HandleUpdate(update, now)
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
	bot := InitTestBot()

	user := TestUser
	user.ID, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: &user,
			Chat: &tgbotapi.Chat{Type: "private"},
		},
	}
	ssauparser.HeadURL = TestServer
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
	ssauparser.HeadURL = TestServer
	bot := InitTestBot()
	bot.Week = 5
	bot.WkPath = os.Getenv("WK_PATH")
	user := database.TgUser{}
	user.ChatID, _ = strconv.ParseInt(os.Getenv("TELEGRAM_TEST_USER"), 0, 64)
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
		img := database.Schedule{
			ScheduleID: sh.SheduleID,
			IsGroup:    sh.IsGroup,
		}
		_, err = func() (tgbotapi.FileBytes, error) {
			var _ time.Time = now

			return htmlschedule.CreateWeekImg(bot.DB, bot.WkPath, &user, img, sh.Week, bot.Week)
		}()
		if err != nil {
			log.Fatal(err)
		}
	}
	t.Log("ok")
}
