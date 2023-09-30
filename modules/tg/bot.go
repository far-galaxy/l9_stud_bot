package tg

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
	"xorm.io/xorm"
)

type Bot struct {
	Name      string
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

var envKeys = []string{
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
	for _, key := range envKeys {
		if _, exists := os.LookupEnv(key); !exists {
			return fmt.Errorf("lost env key: %s", key)
		}
	}

	return nil
}

// Полная инициализация бота со стороны Telegram и БД
func InitBot(db database.DB, token string, build string) (*Bot, error) {
	var bot Bot
	bot.Build = build
	engine, err := database.Connect(db, database.InitLog("sql"))
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
	logger := log.New(database.InitLog("tg"), "", log.LstdFlags)
	err = tgbotapi.SetLogger(logger)
	if err != nil {
		return nil, err
	}
	bot.GetUpdates()

	bot.Name = bot.TG.Self.UserName
	log.Printf("Authorized on account %s", bot.Name)
	bot.Debug = log.New(io.MultiWriter(os.Stderr, database.InitLog("messages")), "", log.LstdFlags)

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

	var tgUser database.TgUser
	if len(users) == 0 {
		l9id, err := database.GenerateID(db)
		if err != nil {
			return nil, err
		}

		user := database.User{
			L9Id: l9id,
		}

		tgUser = database.TgUser{
			L9Id: l9id,
			//Name:   name,
			TgId:   id,
			PosTag: database.NotStarted,
		}
		_, err = db.Insert(user, tgUser)
		if err != nil {
			return nil, err
		}
	} else {
		tgUser = users[0]
	}

	return &tgUser, nil
}

func (bot *Bot) DeleteUser(user database.TgUser) error {
	if _, err := bot.DB.Delete(&user); err != nil {
		return err
	}
	if _, err := bot.DB.Delete(&database.ShedulesInUser{L9Id: user.L9Id}); err != nil {
		return err
	}
	if _, err := bot.DB.Delete(&database.User{L9Id: user.L9Id}); err != nil {
		return err
	}
	if _, err := bot.DB.Delete(&database.File{TgId: user.TgId}); err != nil {
		return err
	}

	return nil
}

func (bot *Bot) HandleUpdate(update tgbotapi.Update, now ...time.Time) (tgbotapi.Message, error) {
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	if update.Message != nil {
		return bot.HandleMessage(update.Message, now[0])
	}
	if update.CallbackQuery != nil {
		return bot.HandleCallback(update.CallbackQuery, now[0])
	}
	if update.InlineQuery != nil {
		return bot.HandleInlineQuery(update)
	}
	if update.MyChatMember != nil {
		return bot.ChatActions(update)
	}

	return nilMsg, nil
}

// TODO: заняться перестановкой функций по файлам

func (bot *Bot) ChatActions(update tgbotapi.Update) (tgbotapi.Message, error) {
	action := update.MyChatMember

	if action.NewChatMember.Status == "member" &&
		action.OldChatMember.Status != "administrator" {
		msg := tgbotapi.NewMessage(
			action.Chat.ID,
			"Привет! Ты меня добавил в чат, и теперь можно смотерть расписание прямо отсюда :)",
		)
		group := database.GroupChatInfo{
			ChatID: action.Chat.ID,
		}
		if _, err := bot.DB.InsertOne(group); err != nil {
			return nilMsg, err
		}

		return bot.TG.Send(msg)
	} else if action.NewChatMember.Status == "administrator" {
		msg := tgbotapi.NewMessage(
			action.Chat.ID,
			"Не думаю, что есть необходимость делать меня администратором\n"+
				"А то вдруг начнётся восстание машин и я случайно поудаляю ваши чаты (:",
		)

		return bot.TG.Send(msg)
	}

	return nilMsg, nil
}

func (bot *Bot) HandleInlineQuery(update tgbotapi.Update) (tgbotapi.Message, error) {
	isGroupChat := update.InlineQuery.ChatType == "group"
	query := update.InlineQuery
	var results []interface{}
	if len(query.Query) < 3 {
		results = append(results, tgbotapi.NewInlineQueryResultArticleHTML(
			"1",
			"Запрос слишком короткий...",
			"Запрос слишком короткий...",
		))

		return bot.SendInlineResult(query.ID, results)
	}

	list, siteErr := ssauparser.SearchInRasp(query.Query)
	if siteErr != nil {
		results = append(results, tgbotapi.NewInlineQueryResultArticleHTML(
			"1",
			"Ошибка на стороне сайта",
			"Ошибка на стороне сайта",
		))

		return bot.SendInlineResult(query.ID, results)
	}

	if len(list) == 0 {
		results = append(results, tgbotapi.NewInlineQueryResultArticleHTML(
			"1",
			"Ничего не найдено ):",
			"Ничего не найдено ):",
		))

		return bot.SendInlineResult(query.ID, results)
	}

	for i, res := range list {
		isGroup := strings.Contains(res.URL, "group")
		var q string
		postfix := ""
		if isGroupChat {
			postfix = fmt.Sprintf("@%s", bot.Name)
		}
		command := "/staff"
		if isGroup {
			command = "/group"
		}
		q = fmt.Sprintf("%s%s %d", command, postfix, res.ID)

		results = append(results, tgbotapi.NewInlineQueryResultArticleHTML(
			fmt.Sprintf("%d", i+1),
			res.Text,
			q,
		))

	}

	return bot.SendInlineResult(query.ID, results)
}

// Отправка результатов Inline-запроса
func (bot *Bot) SendInlineResult(queryID string, results []interface{}) (tgbotapi.Message, error) {
	ans := tgbotapi.InlineConfig{
		InlineQueryID: queryID,
		IsPersonal:    true,
		CacheTime:     0,
		Results:       results,
	}
	_, err := bot.TG.Request(ans)

	return nilMsg, err
}

func (bot *Bot) HandleMessage(msg *tgbotapi.Message, now time.Time) (tgbotapi.Message, error) {
	// Игнорируем "сообщения" о входе в чат
	if len(msg.NewChatMembers) != 0 || msg.LeftChatMember != nil {
		return nilMsg, nil
	}
	if msg.Chat.Type == "group" &&
		len(msg.Entities) != 0 &&
		msg.Entities[0].Type == "bot_command" {

		return bot.HandleGroup(msg, now)
	}
	user, err := InitUser(bot.DB, msg.From)
	if err != nil {
		return nilMsg, err
	}

	bot.Debug.Printf("Message [%d:%d] <%s> %s", user.L9Id, user.TgId, user.Name, msg.Text)
	bot.Messages++
	if strings.Contains(msg.Text, "/help") {
		return bot.SendMsg(user, bot.HelpTxt, bot.AutoGenKeyboard(user))
	}
	if strings.Contains(msg.Text, "/start") && user.PosTag != database.NotStarted {
		if err := bot.DeleteUser(*user); err != nil {
			return nilMsg, err
		}
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
		if msg.Text == "Моё расписание" {
			return bot.GetPersonal(now, user)
		} else if msg.Text == "Настройки" {
			return bot.GetOptions(user)
		} else if strings.Contains(msg.Text, "/keyboard") {
			return bot.SendMsg(
				user,
				"Кнопки действий выданы",
				bot.AutoGenKeyboard(user),
			)
		} else if KeywordContains(msg.Text, []string{"/group", "/staff"}) {
			return bot.GetSheduleFromCmd(now, user, msg.Text)
		} else if KeywordContains(msg.Text, AdminKey) && user.TgId == bot.TestUser {
			return bot.AdminHandle(msg)
		}

		return bot.Find(now, user, msg.Text)
	case database.Add:
		return bot.Find(now, user, msg.Text)
	case database.Set:
		return bot.SetFirstTime(msg, user)
	case database.Delete:
		return bot.DeleteGroup(user, msg.Text)

	default:
		return bot.Etc(user)
	}
}

// Получение расписания из команды /{group, staff} ID_ расписания
func (bot *Bot) GetSheduleFromCmd(
	now time.Time,
	user *database.TgUser,
	query string,
) (
	tgbotapi.Message,
	error,
) {
	isGroup := strings.Contains(query, "/group")
	cmd := strings.Split(query, " ")
	if len(cmd) == 1 {
		return bot.SendMsg(user, "Необходимо указать ID расписания",
			nilKey)
	}
	sheduleID, err := strconv.ParseInt(cmd[1], 10, 64)
	if err != nil {
		return bot.SendMsg(user, "Некорректный ID расписания",
			nilKey)
	}
	shedule := ssauparser.WeekShedule{
		IsGroup:   isGroup,
		SheduleID: sheduleID,
	}
	notExists, _ := ssauparser.CheckGroupOrTeacher(bot.DB, shedule)

	return bot.ReturnSummary(notExists, user.PosTag == database.Add, user, shedule, now)
}

func (bot *Bot) HandleGroup(msg *tgbotapi.Message, now time.Time) (tgbotapi.Message, error) {
	group := database.GroupChatInfo{
		ChatID: msg.Chat.ID,
	}
	if _, err := bot.DB.Get(&group); err != nil {
		return nilMsg, err
	}
	bot.Debug.Printf("ChatCommand [%d] %s", group.ChatID, msg.Text)

	// TODO: добавить игнор команд для групповых чатов в ЛС
	/*
		if strings.Contains(msg.Text, "/add") {
			cmd := strings.Split(msg.Text, " ")
			if len(cmd) == 1 {
				ans := tgbotapi.NewMessage(
					group.ChatID,
					"Необходимо указать запрос прямо в команде!\n"+
						"Например, /add@l9_stud_bot 2405-240502D",
				)

				return bot.TG.Send(ans)
			}
			fakeUser := database.TgUser{
				TgId:   group.ChatID,
				PosTag: database.Ready,
			}

			return bot.Find(now, &fakeUser, cmd[1])
		}*/
	if KeywordContains(msg.Text, []string{"/group", "/staff"}) {
		fakeUser := database.TgUser{
			TgId:   group.ChatID,
			PosTag: database.Ready,
		}

		return bot.GetSheduleFromCmd(now, &fakeUser, msg.Text)
	}
	ans := tgbotapi.NewMessage(msg.Chat.ID, "Исполняю команду")
	ans.ReplyMarkup = CancelKey()

	return bot.TG.Send(ans)
}

func (bot *Bot) HandleCallback(query *tgbotapi.CallbackQuery, now time.Time) (tgbotapi.Message, error) {
	user, err := InitUser(bot.DB, query.From)
	if err != nil {
		return nilMsg, err
	}
	bot.Debug.Printf("Callback [%d:%d] <%s> %s", user.L9Id, user.TgId, user.Name, query.Data)
	bot.Callbacks++
	if query.Data == "cancel" {
		return nilMsg, bot.Cancel(user, query)
	}
	if user.PosTag == database.NotStarted {
		return bot.Start(user)
	} else if user.PosTag == database.Ready || user.PosTag == database.Add {
		if strings.Contains(query.Data, SummaryPrefix) {
			err = bot.HandleSummary(user, query, now)
		} else if strings.Contains(query.Data, "opt") {
			err = bot.HandleOptions(user, query)
		} else {
			err = bot.GetShedule(user, query, now)
		}
	} else {
		return bot.Etc(user)
	}

	// Обработка ошибок
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

	return nilMsg, nil
}

func (bot *Bot) CheckBlocked(err error, user database.TgUser) {
	if !strings.Contains(err.Error(), "blocked by user") {
		if err := bot.DeleteUser(user); err != nil {
			log.Println(err)
		}

		return
	}
	log.Println(err)
}
