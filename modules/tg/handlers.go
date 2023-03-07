package tg

import (
	"log"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func (bot *Bot) Etc() {
	msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Oops!")
	bot.TG.Send(msg)
}
