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
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Начало занятий", bell[options.First]), "opt_first")},
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Следующая пара", bell[options.NextNote]), "opt_lesson")},
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Следующий день", bell[options.NextDay]), "opt_day")},
		{tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Следующая неделя", bell[options.NextWeek]), "opt_week")},
		{tgbotapi.NewInlineKeyboardButtonData("↩ Закрыть", "cancel")},
	}
	if options.First {
		markup = append(markup[:2], markup[1:]...)
		markup[1] = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⏰ Настроить время (%d)", options.FirstTime), "opt_set"),
		}
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
	case "opt_first":
		options.First = !options.First
	case "opt_set":
		user.PosTag = database.Set
		if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
			return err
		}
		txt := fmt.Sprintf(
			"Введи время в минутах, за которое мне надо сообщить о начале занятий\n"+
				"Сейчас установлено %d минут",
			options.FirstTime,
		)
		_, err := bot.EditOrSend(user.TgId, txt, "", tgbotapi.InlineKeyboardMarkup{}, *query.Message)
		return err

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
