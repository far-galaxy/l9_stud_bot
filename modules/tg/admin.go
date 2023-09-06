package tg

import (
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (bot *Bot) Scream(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	var users []database.TgUser
	if err := bot.DB.Where("tgid > 0").Find(&users); err != nil {
		return nilMsg, err
	}
	scream := tgbotapi.NewMessage(
		0,
		strings.TrimPrefix(msg.Text, "/scream"),
	)
	for _, u := range users {
		scream.ChatID = u.TgId
		if _, err := bot.TG.Send(scream); err != nil {
			if !strings.Contains(err.Error(), "blocked by user") {
				bot.Debug.Println(err)
			}
		}
	}
	scream.ChatID = bot.TestUser
	scream.Text = "Сообщения отправлены"
	return bot.TG.Send(scream)
}
