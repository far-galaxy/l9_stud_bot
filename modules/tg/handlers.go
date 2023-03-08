package tg

import (
	"log"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/builder"
)

func (bot *Bot) InitUser(msg *tgbotapi.Message) (*database.TgUser, error) {
	db := &bot.DB
	var users []database.TgUser
	err := db.Find(&users, &database.TgUser{TgId: msg.Chat.ID})
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
			Name:   msg.From.UserName,
			TgId:   msg.Chat.ID,
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

func (bot *Bot) Start() {
	bot.TG_user.PosTag = "add"
	_, err := bot.DB.Update(bot.TG_user)
	if err != nil {
		log.Fatal(err)
	}
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Hello!")
	bot.TG.Send(msg)
}

func (bot *Bot) Find(query string) {
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
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Many groups in base. Please sekect one")
		msg.ReplyMarkup = GenerateKeyboard(GenerateGroupsArray(allGroups), query)
		bot.TG.Send(msg)
	} else if len(allTeachers) != 0 {
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Many teachers in base. Please sekect one")
		msg.ReplyMarkup = GenerateKeyboard(GenerateTeachersArray(allTeachers), query)
		bot.TG.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Nothing found ):")
		bot.TG.Send(msg)
	}

}

func (bot *Bot) Etc() {
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Oops!")
	bot.TG.Send(msg)
}
