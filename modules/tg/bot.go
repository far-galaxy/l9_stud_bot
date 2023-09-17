package tg

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"xorm.io/xorm"
)

type Bot struct {
	TG        *tgbotapi.BotAPI
	DB        *xorm.Engine
	TestUser  int64
	HelpTxt   string
	Week      int
	WkPath    string
	Debug     *log.Logger
	Updates   *tgbotapi.UpdatesChannel
	Messages  int64
	Callbacks int64
	Build     string
}

// TODO: завернуть в структуру
var env_keys = []string{
	"TELEGRAM_APITOKEN",
	"TELEGRAM_TEST_USER",
	"WK_PATH",
	"MYSQL_USER",
	"MYSQL_PASS",
	"MYSQL_DB",
	"START_WEEK",
	"RASP_URL",
	"NOTIFY_PERIOD",
	"SHEDULES_CHECK_PERIOD",
}

func CheckEnv() error {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	for _, key := range env_keys {
		if _, exists := os.LookupEnv(key); !exists {
			return fmt.Errorf("lost env key: %s", key)
		}
	}
	return nil
}

// Полная инициализация бота со стороны Telegram и БД
func InitBot(files database.LogFiles, db database.DB, token string, build string) (*Bot, error) {
	var bot Bot
	bot.Build = build
	engine, err := database.Connect(db, files.DBLogFile)
	if err != nil {
		return nil, err
	}
	//defer engine.Close()
	bot.DB = engine

	bot.TG, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	bot.TG.Debug = true
	//logger := log.New(io.MultiWriter(os.Stdout, database.CreateLog("tg")), "", log.LstdFlags)
	logger := log.New(files.TgLogFile, "", log.LstdFlags)
	err = tgbotapi.SetLogger(logger)
	if err != nil {
		return nil, err
	}
	bot.GetUpdates()

	log.Printf("Authorized on account %s", bot.TG.Self.UserName)
	bot.Debug = log.New(io.MultiWriter(os.Stderr, files.DebugFile), "", log.LstdFlags)

	return &bot, nil
}

func (bot *Bot) GetUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.TG.GetUpdatesChan(u)
	bot.Updates = &updates
}

func (bot *Bot) SendMsg(user *database.TgUser, text string, markup interface{}) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(user.TgId, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = markup
	return bot.TG.Send(msg)
}

// Получение данных о пользователе из БД и создание нового при необходимости
func InitUser(db *xorm.Engine, user *tgbotapi.User) (*database.TgUser, error) {
	id := user.ID
	//name := user.FirstName + " " + user.LastName

	var users []database.TgUser
	err := db.Find(&users, &database.TgUser{TgId: id})
	if err != nil {
		return nil, err
	}

	var tg_user database.TgUser
	if len(users) == 0 {
		l9id, err := database.GenerateID(db)
		if err != nil {
			return nil, err
		}

		user := database.User{
			L9Id: l9id,
		}

		tg_user = database.TgUser{
			L9Id: l9id,
			//Name:   name,
			TgId:   id,
			PosTag: database.NotStarted,
		}
		_, err = db.Insert(user, tg_user)
		if err != nil {
			return nil, err
		}
	} else {
		tg_user = users[0]
	}
	return &tg_user, nil
}

func (bot *Bot) DeleteUser(user database.TgUser) {
	bot.DB.Delete(&user)
	bot.DB.Delete(&database.ShedulesInUser{L9Id: user.L9Id})
	bot.DB.Delete(&database.User{L9Id: user.L9Id})
	bot.DB.Delete(&database.File{TgId: user.TgId})
}

func (bot *Bot) HandleUpdate(update tgbotapi.Update, now ...time.Time) (tgbotapi.Message, error) {
	if update.Message != nil {
		msg := update.Message
		user, err := InitUser(bot.DB, msg.From)
		if err != nil {
			return nilMsg, err
		}
		options := database.ShedulesInUser{
			L9Id: user.L9Id,
		}
		if _, err := bot.DB.Get(&options); err != nil {
			return nilMsg, err
		}
		bot.Debug.Printf("Message [%d:%d] <%s> %s", user.L9Id, user.TgId, user.Name, msg.Text)
		bot.Messages += 1
		if strings.Contains(msg.Text, "/help") {
			return bot.SendMsg(user, bot.HelpTxt, GeneralKeyboard(options.UID != 0))
		}
		if strings.Contains(msg.Text, "/start") && user.PosTag != database.NotStarted {
			bot.DeleteUser(*user)
			if _, err = bot.SendMsg(
				user,
				"Весь прогресс сброшен\nДобро пожаловать снова (:",
				tgbotapi.ReplyKeyboardRemove{RemoveKeyboard: true},
			); err != nil {
				return nilMsg, err
			}
			user, err = InitUser(bot.DB, msg.From)
			if err != nil {
				return nilMsg, err
			}
		}
		switch user.PosTag {
		case database.NotStarted:
			return bot.Start(user)
		case database.Ready:
			if len(now) == 0 {
				now = append(now, msg.Time())
			}
			if msg.Text == "Моё расписание" {
				return bot.GetPersonal(now[0], user)
			} else if msg.Text == "Настройки" {
				return bot.GetOptions(user)
			} else if strings.Contains(msg.Text, "/keyboard") {
				return bot.SendMsg(
					user,
					"Кнопки действий выданы",
					GeneralKeyboard(options.UID != 0),
				)
			} else if KeywordContains(msg.Text, AdminKey) && user.TgId == bot.TestUser {
				return bot.AdminHandle(msg)
			}
			return bot.Find(now[0], user, msg.Text)
		case database.Add:
			return bot.Find(now[0], user, msg.Text)
		case database.Set:
			return bot.SetFirstTime(msg, user)
		case database.Delete:
			return bot.DeleteGroup(user, msg.Text)

		default:
			return bot.Etc(user)
		}
	}
	if update.CallbackQuery != nil {
		query := update.CallbackQuery
		user, err := InitUser(bot.DB, query.From)
		if err != nil {
			return nilMsg, err
		}
		bot.Debug.Printf("Callback [%d:%d] <%s> %s", user.L9Id, user.TgId, user.Name, query.Data)
		bot.Callbacks += 1
		if query.Data == "cancel" {
			return nilMsg, bot.Cancel(user, query)
		}
		if user.PosTag == database.NotStarted {
			return bot.Start(user)
		} else if user.PosTag == database.Ready || user.PosTag == database.Add {
			if strings.Contains(query.Data, SummaryPrefix) {
				err = bot.HandleSummary(user, query, now...)
			} else if strings.Contains(query.Data, "opt") {
				err = bot.HandleOptions(user, query)
			} else {
				err = bot.GetShedule(user, query, now...)
			}
		} else {
			return bot.Etc(user)
		}

		if err != nil {
			if strings.Contains(err.Error(), "message is not modified") {
				callback := tgbotapi.NewCallback(query.ID, "Ничего не изменилось")
				_, err = bot.TG.Request(callback)
				if err != nil {
					return nilMsg, err
				}
				bot.Debug.Println("Message is not modified")
				return nilMsg, nil
			} else if strings.Contains(err.Error(), "no lessons") {
				callback := tgbotapi.NewCallback(query.ID, "Тут занятий уже нет")
				_, err = bot.TG.Request(callback)
				if err != nil {
					return nilMsg, err
				}
				bot.Debug.Println(err)
			}
			return nilMsg, err
		}
	}
	return nilMsg, nil
}
