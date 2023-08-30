package tg

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"xorm.io/xorm"
)

type Bot struct {
	TG       *tgbotapi.BotAPI
	DB       *xorm.Engine
	TestUser int64
	HelpTxt  string
	Week     int
	WkPath   string
	Debug    *log.Logger
	Updates  *tgbotapi.UpdatesChannel
}

var env_keys = []string{
	"TELEGRAM_APITOKEN",
	"TELEGRAM_TEST_USER",
	"WK_PATH",
	"MYSQL_USER",
	"MYSQL_PASS",
	"MYSQL_DB",
	"START_WEEK",
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
func InitBot(files database.LogFiles, db database.DB, token string) (*Bot, error) {
	var bot Bot
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

// Получение данных о пользователе из БД и создание нового при необходимости
func InitUser(db *xorm.Engine, user *tgbotapi.User) (*database.TgUser, error) {
	id := user.ID
	name := user.FirstName + " " + user.LastName

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
			L9Id:   l9id,
			Name:   name,
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

func (bot *Bot) HandleUpdate(update tgbotapi.Update, now ...time.Time) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	if update.Message != nil {
		msg := update.Message
		user, err := InitUser(bot.DB, msg.From)
		if err != nil {
			return nilMsg, err
		}
		bot.Debug.Printf("Message [%d] <%s> %s", user.L9Id, user.Name, msg.Text)
		if strings.Contains(msg.Text, "/help") {
			msg := tgbotapi.NewMessage(user.TgId, bot.HelpTxt)
			return bot.TG.Send(msg)
		}
		switch user.PosTag {
		case database.NotStarted:
			err = bot.Start(user)
		case database.Ready:
			if len(now) == 0 {
				now = append(now, msg.Time())
			}
			if msg.Text == "Моё расписание" {
				return bot.GetPersonal(now[0], user)
			} else if msg.Text == "Настройки" {
				return bot.GetOptions(user)
			} else if strings.Contains(msg.Text, "/keyboard") {
				options := database.ShedulesInUser{
					L9Id: user.L9Id,
				}
				if _, err := bot.DB.Get(&options); err != nil {
					return nilMsg, err
				}
				msg := tgbotapi.NewMessage(user.TgId, "Клавиатура выдана")
				msg.ReplyMarkup = GeneralKeyboard(options.UID != 0)
				return bot.TG.Send(msg)
			} else if strings.Contains(msg.Text, "/scream") && user.TgId == bot.TestUser {
				var users []database.TgUser
				if err := bot.DB.Where("tgid > 0").Find(&users); err != nil {
					return nilMsg, err
				}
				msg := tgbotapi.NewMessage(
					0,
					strings.TrimPrefix(msg.Text, "/scream"),
				)
				for _, u := range users {
					msg.ChatID = u.TgId
					if _, err := bot.TG.Send(msg); err != nil {
						if !strings.Contains(err.Error(), "blocked by user") {
							bot.Debug.Println(err)
						}
					}
				}
				msg.ChatID = bot.TestUser
				msg.Text = "Сообщения отправлены"
				return bot.TG.Send(msg)
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
		if err != nil {
			return nilMsg, err
		}
	}
	if update.CallbackQuery != nil {
		query := update.CallbackQuery
		user, err := InitUser(bot.DB, query.From)
		if err != nil {
			return nilMsg, err
		}
		bot.Debug.Printf("Callback [%d] <%s> %s", user.L9Id, user.Name, query.Data)
		if query.Data == "cancel" {
			return nilMsg, bot.Cancel(user, query)
		}
		if user.PosTag == database.NotStarted {
			err = bot.Start(user)
		} else if user.PosTag == database.Ready || user.PosTag == database.Add {
			if strings.Contains(query.Data, "sh") {
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
			return nilMsg, err
		}
		/*if query.ID != "" {
			callback := tgbotapi.NewCallback(query.ID, query.Data)
			_, err = bot.TG.Request(callback)
			if err != nil {
				return nilMsg, err
			}
		}*/
	}
	return nilMsg, nil
}

func (bot *Bot) DeleteGroup(user *database.TgUser, text string) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	user.PosTag = database.Ready
	if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
		return nilMsg, err
	}
	var msg tgbotapi.MessageConfig
	if strings.ToLower(text) == "да" {
		userInfo := database.ShedulesInUser{
			L9Id: user.L9Id,
		}
		if _, err := bot.DB.Delete(&userInfo); err != nil {
			return nilMsg, err
		}
		files := database.File{
			TgId:       user.L9Id,
			IsPersonal: true,
		}
		if _, err := bot.DB.UseBool("IsPersonal").Delete(&files); err != nil {
			return nilMsg, err
		}
		msg = tgbotapi.NewMessage(user.TgId, "Группа отключена")
		msg.ReplyMarkup = GeneralKeyboard(false)
	} else {
		msg = tgbotapi.NewMessage(user.TgId, "Действие отменено")
	}
	return bot.TG.Send(msg)
}

func (bot *Bot) SetFirstTime(msg *tgbotapi.Message, user *database.TgUser) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	t, err := strconv.Atoi(msg.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(
			user.TgId,
			"Ой, время соообщения о начале занятий введено как-то неверно ):",
		)
		return bot.TG.Send(msg)
	}
	userInfo := database.ShedulesInUser{
		L9Id: user.L9Id,
	}
	if _, err := bot.DB.Get(&userInfo); err != nil {
		return nilMsg, err
	}
	if t <= 10 {
		msg := tgbotapi.NewMessage(
			user.TgId,
			"Ой, установлено слишком малое время. Попробуй ввести большее время",
		)
		return bot.TG.Send(msg)
	} else if t > 240 {
		msg := tgbotapi.NewMessage(
			user.TgId,
			"Ой, установлено слишком большое время. Попробуй ввести меньшее время",
		)
		return bot.TG.Send(msg)
	}
	userInfo.FirstTime = t / 5 * 5
	if _, err := bot.DB.ID(userInfo.UID).Update(userInfo); err != nil {
		return nilMsg, err
	}
	user.PosTag = database.Ready
	if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
		return nilMsg, err
	}
	return bot.GetOptions(user)
}
