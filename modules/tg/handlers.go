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
	db := &bot.DB
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
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Привет! Введи свой номер группы или фамилию преподавателя")
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
			bot.TG_user.PosTag = "confirm_group"
		}
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Вот что я нашёл.\nВыбери свою группу")
		msg.ReplyMarkup = GenerateKeyboard(GenerateGroupsArray(allGroups), query)
		bot.TG.Send(msg)
	} else if len(allTeachers) != 0 {
		if bot.TG_user.PosTag == "add" {
			bot.TG_user.PosTag = "confirm_teacher"
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

func (bot *Bot) Confirm(query *tgbotapi.CallbackQuery, tg_user *database.TgUser, isGroup bool) {
	groupId, err := strconv.ParseInt(query.Data, 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	var groups []database.ShedulesInUser
	err = bot.DB.Find(&groups, &database.ShedulesInUser{
		SheduleId: groupId,
		IsTeacher: !isGroup,
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(groups) == 0 {
		shInUser := database.ShedulesInUser{
			L9Id:      tg_user.L9Id,
			IsTeacher: !isGroup,
			SheduleId: groupId,
		}
		bot.DB.InsertOne(shInUser)
		delete := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
		bot.TG.Request(delete)
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Подключено!")
		bot.TG.Send(msg)

		tg_user.PosTag = "ready"
		bot.DB.Update(tg_user)
	} else {
		callback := tgbotapi.NewCallback(query.ID, "Эта группа уже подключена!")
		bot.TG.Request(callback)
	}
}

func (bot *Bot) Etc() {
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Oй!")
	bot.TG.Send(msg)
}
