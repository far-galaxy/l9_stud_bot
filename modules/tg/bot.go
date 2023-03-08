package tg

import (
	"log"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/xorm"
)

type Bot struct {
	TG      *tgbotapi.BotAPI
	DB      xorm.Engine
	TG_user database.TgUser
}

func (bot *Bot) InitBot(token string, engine xorm.Engine) error {
	var err error
	bot.TG, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}
	bot.TG.Debug = true

	bot.DB = engine

	log.Printf("Authorized on account %s", bot.TG.Self.UserName)
	return nil
}

func (bot *Bot) GetUpdates() *tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.TG.GetUpdatesChan(u)
	return &updates
}
