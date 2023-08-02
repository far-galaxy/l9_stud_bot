package tg

import (
	"strconv"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/builder"
)

func (bot *Bot) Start() error {
	bot.TG_user.PosTag = "ready"
	_, err := bot.DB.Update(bot.TG_user)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(
		bot.TG_user.TgId,
		`Привет! Я новая версия этого бота и пока тут за главного (:
У меня можно посмотреть в удобном формате <b>ближайшие пары</b>, расписание <b>по дням</b> и даже <b>по неделям</b>!
Просто напиши мне <b>номер группы</b> или <b>фамилию преподавателя</b>`)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err = bot.TG.Send(msg)
	return err
}

func (bot *Bot) Find(query string) error {
	var groups []database.Group
	bot.DB.Where(builder.Like{"GroupName", query}).Find(&groups)

	var teachers []database.Teacher
	bot.DB.Where(builder.Like{"LastName", query}).Find(&teachers)

	list, _ := ssau_parser.SearchInRasp(query)

	allGroups := groups
	allTeachers := teachers

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
				name := strings.Split(elem.Text, " ")
				allTeachers = append(allTeachers, database.Teacher{
					TeacherId: elem.Id,
					LastName:  name[0],
					FirstName: name[1],
				})
			}
		}
	}

	if len(allGroups) == 1 || len(allTeachers) == 1 {
		if bot.TG_user.PosTag == "add" {
			msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Подключено!")
			keyboard := tgbotapi.NewReplyKeyboard([]tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton("Главное меню")})
			msg.ReplyMarkup = keyboard
			bot.TG.Send(msg)
		} else {
			var sheduleId int64
			var isGroup bool
			if len(allGroups) == 1 {
				sheduleId = allGroups[0].GroupId
				isGroup = true
			} else {
				sheduleId = allTeachers[0].TeacherId
				isGroup = false
			}
			shedule := database.ShedulesInUser{
				IsTeacher: !isGroup,
				SheduleId: sheduleId,
			}
			err := bot.GetSummary([]database.ShedulesInUser{shedule}, false)
			if err != nil {
				return err
			}
		}
		bot.TG_user.PosTag = "ready"
		err := bot.UpdateUserDB()
		return err
	} else if len(allGroups) != 0 {
		if bot.TG_user.PosTag == "add" {
			bot.TG_user.PosTag = "confirm_add_group"
		} else {
			bot.TG_user.PosTag = "confirm_see_group"
		}
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Вот что я нашёл.\nВыбери свою группу")
		msg.ReplyMarkup = GenerateKeyboard(GenerateGroupsArray(allGroups), query)
		bot.TG.Send(msg)
	} else if len(allTeachers) != 0 {
		if bot.TG_user.PosTag == "add" {
			bot.TG_user.PosTag = "confirm_add_teacher"
		} else {
			bot.TG_user.PosTag = "confirm_see_teacher"
		}
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Вот что я нашёл.\nВыбери нужного преподавателя")
		msg.ReplyMarkup = GenerateKeyboard(GenerateTeachersArray(allTeachers), query)
		bot.TG.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "К сожалению, я ничего не нашёл ):\nПроверь свой запрос")
		bot.TG.Send(msg)
	}

	_, err := bot.DB.Update(bot.TG_user)
	return err
}

func (bot *Bot) SeeShedule(query *tgbotapi.CallbackQuery) error {
	isGroup := bot.TG_user.PosTag == "confirm_see_group"
	groupId, err := strconv.ParseInt(query.Data, 0, 64)
	if err != nil {
		return err
	}
	shedule := database.ShedulesInUser{
		IsTeacher: !isGroup,
		SheduleId: groupId,
	}
	err = bot.GetSummary([]database.ShedulesInUser{shedule}, false)
	if err != nil {
		return err
	}
	bot.TG_user.PosTag = "ready"
	err = bot.UpdateUserDB()
	return err
}

func (bot *Bot) HandleSummary(query *tgbotapi.CallbackQuery) error {
	data := strings.Split(query.Data, "_")
	shedule, dt, err := ParseQuery(data)
	if err != nil {
		return err
	}
	if data[1] == "personal" {
		switch data[0] {
		case "day":
			bot.GetPersonalDaySummary(int(dt), *query.Message)
		case "week":
			bot.GetPersonalWeekSummary(int(dt), *query.Message)
		default:
			bot.GetPersonalSummary(*query.Message)
		}
	} else {
		switch data[0] {
		case "day":
			bot.GetDaySummary(shedule, dt, false, *query.Message)
		case "week":
			bot.GetWeekSummary(shedule, dt, false, *query.Message)
		default:
			bot.GetSummary(shedule, false, *query.Message)
		}
	}
	return nil
}

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

func (bot *Bot) Cancel(query *tgbotapi.CallbackQuery) error {
	bot.TG_user.PosTag = "ready"
	err := bot.UpdateUserDB()
	if err != nil {
		return err
	}
	bot.DeleteMsg(query)

	return nil
}

func (bot *Bot) Etc() {
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Oй!")
	bot.TG.Send(msg)
}
