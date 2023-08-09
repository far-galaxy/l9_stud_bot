package tg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/builder"
)

// Приветственное сообщение
func (bot *Bot) Start(user *database.TgUser) error {
	user.PosTag = database.Ready
	_, err := bot.DB.ID(user.L9Id).Update(user)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(
		user.TgId,
		`Привет! У меня можно посмотреть в удобном формате <b>ближайшие пары</b>, расписание <b>по дням</b> и даже <b>по неделям</b>!
Просто напиши мне <b>номер группы</b> или <b>фамилию преподавателя</b>`)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = GeneralKeyboard()
	_, err = bot.TG.Send(msg)
	return err
}

// Поиск расписания по запросу
func (bot *Bot) Find(now time.Time, user *database.TgUser, query string) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	// Поиск в БД
	var groups []database.Group
	if err := bot.DB.Where(builder.Like{"GroupName", query}).Find(&groups); err != nil {
		return nilMsg, err
	}

	var teachers []database.Teacher
	if err := bot.DB.Where(builder.Like{"FirstName", query}).Find(&teachers); err != nil {
		return nilMsg, err
	}

	// Поиск на сайте
	list, siteErr := ssau_parser.SearchInRasp(query)

	allGroups := groups
	allTeachers := teachers

	// Добавляем результаты поиска на сайте к результатам из БД
	for _, elem := range list {
		if strings.Contains(elem.Url, "group") {
			exists := false
			for _, group := range groups {
				if elem.Id == group.GroupId {
					exists = true
					break
				}
			}
			if !exists {
				allGroups = append(allGroups, database.Group{GroupId: elem.Id, GroupName: elem.Text})
			}
		}
		if strings.Contains(elem.Url, "staff") {
			exists := false
			for _, teacher := range teachers {
				if elem.Id == teacher.TeacherId {
					exists = true
					break
				}
			}
			if !exists {
				teacher := ssau_parser.ParseTeacherName(elem.Text)
				teacher.TeacherId = elem.Id
				allTeachers = append(allTeachers, teacher)
			}
		}
	}

	// Если получен единственный результат, сразу выдать (подключить) расписание
	if len(allGroups) == 1 || len(allTeachers) == 1 {
		var sheduleId int64
		var isGroup bool
		if len(allGroups) == 1 {
			sheduleId = allGroups[0].GroupId
			isGroup = true
		} else {
			sheduleId = allTeachers[0].TeacherId
			isGroup = false
		}
		shedule := ssau_parser.WeekShedule{
			IsGroup:   isGroup,
			SheduleId: sheduleId,
		}
		not_exists, _ := ssau_parser.CheckGroupOrTeacher(bot.DB, shedule)
		if not_exists {
			msg := tgbotapi.NewMessage(user.TgId, "Загружаю расписание...\nЭто займёт некоторое время")
			Smsg, _ := bot.TG.Send(msg)
			err := bot.LoadShedule(shedule)
			if err != nil {
				return nilMsg, err
			}
			del := tgbotapi.NewDeleteMessage(Smsg.Chat.ID, Smsg.MessageID)
			if _, err := bot.TG.Request(del); err != nil {
				return nilMsg, err
			}
		}
		// TODO: проверять подключенные ранее расписания
		if user.PosTag == database.Add {
			sh := Swap(shedule)
			sh.L9Id = user.L9Id
			if _, err := bot.DB.InsertOne(&sh); err != nil {
				return nilMsg, err
			}
			user.PosTag = database.Ready
			if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
				return nilMsg, err
			}
			msg := tgbotapi.NewMessage(
				user.TgId,
				"Расписание успешно подключено!\n"+
					"Теперь его можно открыть по кнопке <b>Моё расписание</b>👇",
			)
			msg.ParseMode = tgbotapi.ModeHTML
			msg.ReplyMarkup = GeneralKeyboard()
			return bot.TG.Send(msg)
		} else {
			return bot.GetSummary(now, user, []database.ShedulesInUser{Swap(shedule)}, false)
		}

		// Если получено несколько групп
	} else if len(allGroups) != 0 {
		msg := tgbotapi.NewMessage(user.TgId, "Вот что я нашёл\nВыбери нужную группу")
		msg.ReplyMarkup = GenerateKeyboard(GenerateGroupsArray(allGroups, user.PosTag == database.Add))
		return bot.TG.Send(msg)
		// Если получено несколько преподавателей
	} else if len(allTeachers) != 0 {
		msg := tgbotapi.NewMessage(user.TgId, "Вот что я нашёл\nВыбери нужного преподавателя")
		msg.ReplyMarkup = GenerateKeyboard(GenerateTeachersArray(allTeachers, user.PosTag == database.Add))
		return bot.TG.Send(msg)
		// Если ничего не получено
	} else {
		var msg tgbotapi.MessageConfig
		if siteErr != nil {
			msg = tgbotapi.NewMessage(
				user.TgId,
				"К сожалению, у меня ничего не нашлось, а на сайте ssau.ru/rasp произошла какая-то ошибка :(\n"+
					"Повтори попытку позже",
			)
			bot.Debug.Printf("sasau error: %s", siteErr)
		} else {
			msg = tgbotapi.NewMessage(
				user.TgId,
				"К сожалению, я ничего не нашёл ):\nПроверь свой запрос",
			)
		}

		return bot.TG.Send(msg)
	}
}

// Получить расписание из кнопки
func (bot *Bot) GetShedule(user *database.TgUser, query *tgbotapi.CallbackQuery, now ...time.Time) error {
	data := strings.Split(query.Data, "_")
	if len(data) != 3 {
		return fmt.Errorf("wrong button format: %s", query.Data)
	}
	isGroup := data[1] == "group"
	isAdd := data[0] == "true"
	groupId, err := strconv.ParseInt(data[2], 0, 64)
	if err != nil {
		return err
	}
	shedule := ssau_parser.WeekShedule{
		IsGroup:   isGroup,
		SheduleId: groupId,
	}
	not_exists, _ := ssau_parser.CheckGroupOrTeacher(bot.DB, shedule)
	if not_exists {
		msg := tgbotapi.NewMessage(user.TgId, "Загружаю расписание...\nЭто займёт некоторое время")
		Smsg, _ := bot.TG.Send(msg)
		if err := bot.LoadShedule(shedule); err != nil {
			return err
		}
		del := tgbotapi.NewDeleteMessage(Smsg.Chat.ID, Smsg.MessageID)
		if _, err := bot.TG.Request(del); err != nil {
			return err
		}

	}
	if !isAdd {
		if len(now) == 0 {
			now = append(now, time.Now())
		}
		_, err = bot.GetSummary(now[0], user, []database.ShedulesInUser{Swap(shedule)}, false, *query.Message)
	} else {
		sh := Swap(shedule)
		sh.L9Id = user.L9Id
		if _, err = bot.DB.InsertOne(&sh); err != nil {
			return err
		}
		user.PosTag = database.Ready
		if _, err = bot.DB.ID(user.L9Id).Update(user); err != nil {
			return err
		}
		_, err = bot.GetPersonal(now[0], user)
	}
	return err
}

func (bot *Bot) HandleSummary(user *database.TgUser, query *tgbotapi.CallbackQuery, now ...time.Time) error {
	data := strings.Split(query.Data, "_")
	shedule, dt, err := ParseQuery(data)
	if err != nil {
		return err
	}
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	if data[2] == "personal" {
		switch data[1] {
		case "day":
			var shedules []database.ShedulesInUser
			bot.DB.ID(user.L9Id).Find(&shedules)
			_, err = bot.GetDaySummary(now[0], user, shedules, dt, true, *query.Message)
		case "week":
			err = bot.GetWeekSummary(now[0], user, shedule[0], dt, true, *query.Message)
		default:
			_, err = bot.GetPersonal(now[0], user, *query.Message)
		}
	} else {
		switch data[1] {
		case "day":
			_, err = bot.GetDaySummary(now[0], user, shedule, dt, false, *query.Message)
		case "week":
			err = bot.GetWeekSummary(now[0], user, shedule[0], dt, false, *query.Message)

		default:
			_, err = bot.GetSummary(now[0], user, shedule, false, *query.Message)
		}
	}
	return err
}

/*
	func (bot *Bot) Confirm(query *tgbotapi.CallbackQuery) error {
		isGroup := bot.TG_user.PosTag == "confirm_add_group"
		groupId, err := strconv.ParseInt(query.Data, 0, 64)
		if err != nil {
			return err
		}
		var groups []database.ShedulesInUser
		err = bot.DB.Find(&groups, &database.ShedulesInUser{
			L9Id:      bot.TG_user.L9Id,
			SheduleId: groupId,
			IsTeacher: !isGroup,
		})
		if err != nil {
			return err
		}
		if len(groups) == 0 {
			shInUser := database.ShedulesInUser{
				L9Id:      bot.TG_user.L9Id,
				IsTeacher: !isGroup,
				SheduleId: groupId,
			}
			bot.DB.InsertOne(shInUser)
			bot.DeleteMsg(query)
			msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Подключено!")
			keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton("Главное меню")})
			msg.ReplyMarkup = keyboard
			bot.TG.Send(msg)

			bot.TG_user.PosTag = "ready"
			err = bot.UpdateUserDB()
			if err != nil {
				return err
			}
		} else {
			var msg string
			if isGroup {
				msg = "Эта группа уже подключена!"
			} else {
				msg = "Этот преподаватель уже подключен!"
			}
			callback := tgbotapi.NewCallback(query.ID, msg)
			bot.TG.Request(callback)
		}
		return nil
	}
*/

func (bot *Bot) Etc(user *database.TgUser) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(user.TgId, "Oй!")
	return bot.TG.Send(msg)
}

func (bot *Bot) Cancel(user *database.TgUser, query *tgbotapi.CallbackQuery) error {
	user.PosTag = database.Ready
	_, err := bot.DB.ID(user.L9Id).Update(user)
	if err != nil {
		return err
	}
	if query.ID != "" {
		callback := tgbotapi.NewCallback(query.ID, "Действие отменено")
		_, err = bot.TG.Request(callback)
		if err != nil {
			return err
		}
	}
	delete := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
	_, err = bot.TG.Request(delete)
	return err
}
