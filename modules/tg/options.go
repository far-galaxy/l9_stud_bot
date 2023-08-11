package tg

import (
	"fmt"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var bell = map[bool]string{true: "🔔", false: "🔕"}
var optStr = "Настройки уведомлений\nНажми на кнопку, чтобы переключить параметр"

func (bot *Bot) GetOptions(user *database.TgUser) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	options := database.ShedulesInUser{
		L9Id: user.L9Id,
	}
	if _, err := bot.DB.Get(&options); err != nil {
		return nilMsg, err
	}
	markup := OptMarkup(options)
	msg := tgbotapi.NewMessage(user.TgId, optStr)
	msg.ReplyMarkup = markup
	return bot.TG.Send(msg)
}

func OptMarkup(options database.ShedulesInUser) tgbotapi.InlineKeyboardMarkup {
	markup := [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Следующая пара", bell[options.NextNote]), "opt_lesson")},
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Следующий день", bell[options.NextDay]), "opt_day")},
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Следующая неделя", bell[options.NextWeek]), "opt_week")},
	}
	return tgbotapi.NewInlineKeyboardMarkup(markup...)
}

func (bot *Bot) HandleOptions(user *database.TgUser, query *tgbotapi.CallbackQuery) error {
	options := database.ShedulesInUser{
		L9Id: user.L9Id,
	}
	if _, err := bot.DB.Get(&options); err != nil {
		return err
	}
	switch query.Data {
	case "opt_lesson":
		options.NextNote = !options.NextNote
	case "opt_day":
		options.NextDay = !options.NextDay
	case "opt_week":
		options.NextWeek = !options.NextWeek
	}
	if _, err := bot.DB.UseBool().ID(options.UID).Update(&options); err != nil {
		return err
	}
	_, err := bot.EditOrSend(user.TgId, optStr, "", OptMarkup(options), *query.Message)
	return err
}
