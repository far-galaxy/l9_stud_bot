package tg

import (
	"fmt"
	"strconv"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GenerateGroupsArray(groups []database.Group) []tgbotapi.InlineKeyboardButton {
	var grKeys []tgbotapi.InlineKeyboardButton
	for _, gr := range groups {
		grKeys = append(grKeys, tgbotapi.NewInlineKeyboardButtonData(gr.GroupName, strconv.FormatInt(gr.GroupId, 10)))
	}
	return grKeys
}

func GenerateName(t database.Teacher) string {
	var initials string
	for _, n := range strings.Split(t.FirstName, " ") {
		initials += fmt.Sprintf("%s.", n[:2])
	}
	name := fmt.Sprintf("%s %s", t.LastName, initials)
	return name
}

func GenerateTeachersArray(teachers []database.Teacher) []tgbotapi.InlineKeyboardButton {
	var teacherKeys []tgbotapi.InlineKeyboardButton
	for _, t := range teachers {
		name := GenerateName(t)
		teacherKeys = append(teacherKeys, tgbotapi.NewInlineKeyboardButtonData(name, strconv.FormatInt(t.TeacherId, 10)))
	}
	return teacherKeys
}

func GenerateKeyboard(array []tgbotapi.InlineKeyboardButton, query string) tgbotapi.InlineKeyboardMarkup {
	var keys []tgbotapi.InlineKeyboardButton
	var markup [][]tgbotapi.InlineKeyboardButton

	for _, key := range array {
		keys = append(keys, key)
		if len(keys) >= 3 {
			markup = append(markup, keys)
			keys = []tgbotapi.InlineKeyboardButton{}
		}
	}
	markup = append(markup, keys)
	no_one := tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel")
	markup = append(markup, []tgbotapi.InlineKeyboardButton{no_one})
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup}
}

func (bot *Bot) DeleteMsg(query *tgbotapi.CallbackQuery) {
	delete := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
	bot.TG.Request(delete)
}

func (bot *Bot) UpdateUserDB() error {
	_, err := bot.DB.ID(bot.TG_user.L9Id).Update(bot.TG_user)
	return err
}
