package tg

import (
	"log"
	"strconv"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/builder"
)

func (bot *Bot) InitUser(id int64, name string) (*database.TgUser, error) {
	db := bot.DB
	var users []database.TgUser
	err := db.Find(&users, &database.TgUser{TgId: id})
	if err != nil {
		log.Fatal(err)
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
			PosTag: "not_started",
		}
		_, err = db.Insert(user, tg_user)
		if err != nil {
			return nil, err
		}
	} else {
		tg_user = users[0]
	}
	bot.TG_user = tg_user
	return &tg_user, nil
}

func (bot *Bot) Start() error {
	bot.TG_user.PosTag = "add"
	_, err := bot.DB.Update(bot.TG_user)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Привет! Введи свой <b>номер группы</b> или <b>фамилию преподавателя</b>")
	msg.ParseMode = tgbotapi.ModeHTML
	_, err = bot.TG.Send(msg)
	return err
}

func (bot *Bot) Find(query string) error {
	var groups []database.Group
	bot.DB.Where(builder.Like{"GroupName", query}).Find(&groups)

	var teachers []database.Teacher
	bot.DB.Where(builder.Like{"LastName", query}).Find(&teachers)

	list, _ := ssau_parser.FindInRasp(query)

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
					MidName:   name[2],
				})
			}
		}
	}

	if len(allGroups) != 0 {
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
	shedules, err := bot.DB.Count(&database.ShedulesInUser{
		L9Id: bot.TG_user.L9Id,
	})
	if err != nil {
		log.Fatal(err)
	}
	if shedules == 0 {
		bot.TG_user.PosTag = "add"
		err = bot.UpdateUserDB()
		if err != nil {
			return err
		}
		bot.DeleteMsg(query)
		msg := tgbotapi.NewMessage(
			bot.TG_user.TgId,
			"Ой, для работы с ботом нужно подключить хотя бы одно расписание группы или преподавателя!\nВведи свой номер группы или фамилию преподавателя",
		)
		bot.TG.Send(msg)
	} else {
		bot.TG_user.PosTag = "ready"
		err = bot.UpdateUserDB()
		if err != nil {
			return err
		}
		bot.DeleteMsg(query)
	}
	return nil
}

func (bot *Bot) Etc() {
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Oй!")
	bot.TG.Send(msg)
}
