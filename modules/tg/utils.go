package tg

import (
	"fmt"
	"strconv"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Создание ряда кнопок из списка групп
func GenerateGroupsArray(groups []database.Group, isAdd bool) []tgbotapi.InlineKeyboardButton {
	var grKeys []tgbotapi.InlineKeyboardButton
	for _, gr := range groups {
		grKeys = append(grKeys, tgbotapi.NewInlineKeyboardButtonData(
			gr.GroupName,
			fmt.Sprintf("%t_group_%d", isAdd, gr.GroupId),
		))
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

// Создание ряда кнопок из списка преподавателей
func GenerateTeachersArray(teachers []database.Teacher, isAdd bool) []tgbotapi.InlineKeyboardButton {
	var teacherKeys []tgbotapi.InlineKeyboardButton
	for _, t := range teachers {
		name := fmt.Sprintf("%s %s", t.FirstName, t.ShortName)
		teacherKeys = append(teacherKeys, tgbotapi.NewInlineKeyboardButtonData(
			name,
			fmt.Sprintf("%t_staff_%d", isAdd, t.TeacherId),
		))
	}
	return teacherKeys
}

// Создание полноценной клавиатуры выбора
func GenerateKeyboard(array []tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
	var keys []tgbotapi.InlineKeyboardButton
	var markup [][]tgbotapi.InlineKeyboardButton
	// Разбиваем список кнопок в ряды по 3 кнопки
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

func SummaryKeyboard(clickedButton string, sheduleId int64, isGroup bool, dt int) tgbotapi.InlineKeyboardMarkup {
	tail := GenerateButtonTail(sheduleId, 0, isGroup)

	near := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Краткая сводка", "sh_near"+tail),
	}
	day := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("День", "sh_day"+tail),
	}
	week := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Неделя", "sh_week"+tail),
	}

	update := GenerateButtonTail(sheduleId, dt, isGroup)
	var arrows []tgbotapi.InlineKeyboardButton
	if clickedButton == "sh_day" || clickedButton == "sh_week" {
		prev_arrow := GenerateButtonTail(sheduleId, dt-1, isGroup)
		next_arrow := GenerateButtonTail(sheduleId, dt+1, isGroup)
		arrows = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⏮", clickedButton+prev_arrow),
			tgbotapi.NewInlineKeyboardButtonData("🔄", clickedButton+update),
			tgbotapi.NewInlineKeyboardButtonData("⏭", clickedButton+next_arrow),
		}
	} else {
		arrows = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔄", clickedButton+update),
		}
	}
	/*options := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Настройки", "options"),
	}*/

	var markup [][]tgbotapi.InlineKeyboardButton
	switch clickedButton {
	case "sh_day":
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, near, week,
		}
	case "sh_week":
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, near, day,
		}
	default:
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, day, week,
		}
	}
	/*if sheduleId == 0 {
		markup = append(markup, options)
	}*/
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup}
}

func GenerateButtonTail(sheduleId int64, dt int, isGroup bool) string {
	var tail string
	if sheduleId == 0 {
		tail = fmt.Sprintf("_personal_%d_0", dt)
	} else if !isGroup {
		tail = fmt.Sprintf("_teacher_%d_%d", dt, sheduleId)
	} else {
		tail = fmt.Sprintf("_group_%d_%d", dt, sheduleId)
	}
	return tail
}

func (bot *Bot) EditOrSend(id int64, str string, markup tgbotapi.InlineKeyboardMarkup, editMsg ...tgbotapi.Message) {
	if len(editMsg) > 0 {
		msg := tgbotapi.NewEditMessageText(
			editMsg[0].Chat.ID,
			editMsg[0].MessageID,
			str,
		)
		msg.ReplyMarkup = &markup
		bot.TG.Request(msg)
	} else {
		msg := tgbotapi.NewMessage(id, str)
		//msg.ReplyMarkup = &markup
		_, err := bot.TG.Send(msg)
		if err != nil {
			bot.Debug.Println(err)
		}
	}
}

func ParseQuery(data []string) ([]database.ShedulesInUser, int, error) {
	isGroup := data[2] == "group"
	sheduleId, err := strconv.ParseInt(data[4], 0, 64)
	if err != nil {
		return nil, 0, err
	}
	shedule := database.ShedulesInUser{
		IsGroup:   !isGroup,
		SheduleId: sheduleId,
	}
	dt, err := strconv.ParseInt(data[3], 0, 0)
	if err != nil {
		return nil, 0, err
	}
	return []database.ShedulesInUser{shedule}, int(dt), nil
}

var SumKey = []string{"near", "day", "week"}

func KeywordContains(str string, keywords []string) bool {
	for _, key := range keywords {
		if strings.Contains(str, key) {
			return true
		}
	}
	return false
}

func (bot *Bot) DeleteMsg(query *tgbotapi.CallbackQuery) {
	delete := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
	bot.TG.Request(delete)
}

// Меняем шило на мыло
func Swap(sh ssau_parser.WeekShedule) database.ShedulesInUser {
	return database.ShedulesInUser{
		IsGroup:   sh.IsGroup,
		SheduleId: sh.SheduleId,
	}
}
