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

func SummaryKeyboard(clickedButton string, sheduleId int64, isTeacher bool, dt int) tgbotapi.InlineKeyboardMarkup {
	tail := GenerateButtonTail(sheduleId, dt, isTeacher)

	near := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Краткая сводка", "near"+tail),
	}
	day := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("День", "day"+tail),
	}
	week := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Неделя", "week"+tail),
	}

	var arrows []tgbotapi.InlineKeyboardButton
	if clickedButton == "day" || clickedButton == "week" {
		prev_arrow := GenerateButtonTail(sheduleId, dt-1, isTeacher)
		next_arrow := GenerateButtonTail(sheduleId, dt+1, isTeacher)
		arrows = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("<", clickedButton+prev_arrow),
			tgbotapi.NewInlineKeyboardButtonData(">", clickedButton+next_arrow),
		}
	}
	/*options := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Настройки", "options"),
	}*/

	var markup [][]tgbotapi.InlineKeyboardButton
	switch clickedButton {
	case "day":
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, near, week,
		}
	case "week":
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, near, day,
		}
	default:
		markup = [][]tgbotapi.InlineKeyboardButton{
			day, week,
		}
	}
	/*if sheduleId == 0 {
		markup = append(markup, options)
	}*/
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup}
}

func GenerateButtonTail(sheduleId int64, dt int, isTeacher bool) string {
	var tail string
	if sheduleId == 0 {
		tail = fmt.Sprintf("_personal_%d", dt)
	} else if isTeacher {
		tail = fmt.Sprintf("_teacher_%d_%d", dt, sheduleId)
	} else {
		tail = fmt.Sprintf("_group_%d_%d", dt, sheduleId)
	}
	return tail
}

func (bot *Bot) EditOrSend(str string, markup tgbotapi.InlineKeyboardMarkup, editMsg ...tgbotapi.Message) {
	if len(editMsg) > 0 {
		msg := tgbotapi.NewEditMessageText(
			editMsg[0].Chat.ID,
			editMsg[0].MessageID,
			str,
		)
		msg.ReplyMarkup = &markup
		bot.TG.Request(msg)
	} else {
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, str)
		msg.ReplyMarkup = markup
		bot.TG.Send(msg)
	}
}

func ParseQuery(data []string) ([]database.ShedulesInUser, int, error) {
	isGroup := data[1] == "group"
	sheduleId, err := strconv.ParseInt(data[3], 0, 64)
	if err != nil {
		return nil, 0, err
	}
	shedule := database.ShedulesInUser{
		IsTeacher: !isGroup,
		SheduleId: sheduleId,
	}
	dt, err := strconv.ParseInt(data[2], 0, 0)
	if err != nil {
		return nil, 0, err
	}
	return []database.ShedulesInUser{shedule}, int(dt), nil
}

func (bot *Bot) DeleteMsg(query *tgbotapi.CallbackQuery) {
	delete := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
	bot.TG.Request(delete)
}

func (bot *Bot) UpdateUserDB() error {
	_, err := bot.DB.ID(bot.TG_user.L9Id).Update(bot.TG_user)
	return err
}
