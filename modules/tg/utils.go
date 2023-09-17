package tg

import (
	"fmt"
	"strconv"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Основные кнопки действий: "Моё расписание" и "Настройки" (опционально)
func GeneralKeyboard(options bool) tgbotapi.ReplyKeyboardMarkup {
	keyboard := [][]tgbotapi.KeyboardButton{{
		tgbotapi.NewKeyboardButton("Моё расписание"),
	}}
	if options {
		keyboard = append(keyboard, []tgbotapi.KeyboardButton{tgbotapi.NewKeyboardButton("Настройки")})
	}
	key := tgbotapi.NewReplyKeyboard(keyboard...)
	key.ResizeKeyboard = true

	return key
}

// Inline-кнопка отмены
func CancelKey() tgbotapi.InlineKeyboardMarkup {
	markup := [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel")},
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup}
}

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

// Создать имя преподавателя формата Фамилия И.О.
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
	noOne := tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel")
	markup = append(markup, []tgbotapi.InlineKeyboardButton{noOne})

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup}
}

type SummaryType string

const (
	SummaryPrefix string      = "sh_"
	Near          SummaryType = "near"
	Day           SummaryType = "day"
	Week          SummaryType = "week"
	ICS           SummaryType = "ics"
)

// Inline-клавиатура карточки с расписанием
func SummaryKeyboard(
	clickedButton SummaryType,
	shedule database.ShedulesInUser,
	isPersonal bool,
	dt int,
) tgbotapi.InlineKeyboardMarkup {
	var sheduleID int64
	if isPersonal {
		sheduleID = 0
	} else {
		sheduleID = shedule.SheduleId
	}
	tail := GenerateButtonTail(sheduleID, 0, shedule.IsGroup)

	near := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			"Краткая сводка",
			SummaryPrefix+string(Near)+tail,
		),
	}
	day := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			"День",
			SummaryPrefix+string(Day)+tail,
		),
	}
	week := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			"Неделя",
			SummaryPrefix+string(Week)+tail,
		),
	}

	update := GenerateButtonTail(sheduleID, dt, shedule.IsGroup)
	ics := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			"🗓 Скачать .ics",
			SummaryPrefix+string(ICS)+update,
		),
	}

	var arrows []tgbotapi.InlineKeyboardButton
	if clickedButton == Day || clickedButton == Week {
		prevArrow := GenerateButtonTail(sheduleID, dt-1, shedule.IsGroup)
		nextArrow := GenerateButtonTail(sheduleID, dt+1, shedule.IsGroup)
		arrows = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⏮", SummaryPrefix+string(clickedButton)+prevArrow),
			tgbotapi.NewInlineKeyboardButtonData("🔄", SummaryPrefix+string(clickedButton)+update),
			tgbotapi.NewInlineKeyboardButtonData("⏭", SummaryPrefix+string(clickedButton)+nextArrow),
		}
	} else {
		arrows = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔄", SummaryPrefix+string(clickedButton)+update),
		}
	}

	var markup [][]tgbotapi.InlineKeyboardButton
	switch clickedButton {
	case Day:
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, near, week,
		}
	case Week:
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, ics, day, near,
		}
	default:
		markup = [][]tgbotapi.InlineKeyboardButton{
			arrows, day, week,
		}
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup}
}

func GenerateButtonTail(sheduleID int64, dt int, isGroup bool) string {
	var tail string
	if sheduleID == 0 {
		tail = fmt.Sprintf("_personal_%d_0", dt)
	} else if !isGroup {
		tail = fmt.Sprintf("_teacher_%d_%d", dt, sheduleID)
	} else {
		tail = fmt.Sprintf("_group_%d_%d", dt, sheduleID)
	}

	return tail
}

// Отправка сообщения или его редактирование, если в editMsg указано сообщение
// TODO: Обрабатывать старые сообщения, которые уже нельзя редактировать (message can't be deleted for everyone)
func (bot *Bot) EditOrSend(
	id int64,
	str string,
	imageID string,
	markup tgbotapi.InlineKeyboardMarkup,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {

	if len(editMsg) > 0 {
		// Редактируем
		if imageID != "" {
			// Обновляем фото, если есть
			// TODO: реализовать нормальное обновление фото, когда нужный метод появится в tgbotapi
			del := tgbotapi.NewDeleteMessage(
				editMsg[0].Chat.ID,
				editMsg[0].MessageID,
			)
			if _, err := bot.TG.Request(del); err != nil {
				return nilMsg, err
			}
			newMsg := tgbotapi.NewPhoto(
				editMsg[0].Chat.ID,
				tgbotapi.FileID(imageID),
			)
			newMsg.Caption = str
			newMsg.ParseMode = tgbotapi.ModeHTML
			if len(markup.InlineKeyboard) != 0 {
				newMsg.ReplyMarkup = &markup
			}

			return bot.TG.Send(newMsg)
		} else if len(editMsg[0].Photo) == 0 {
			// Фото нет и не было, только текст
			msg := tgbotapi.NewEditMessageText(
				editMsg[0].Chat.ID,
				editMsg[0].MessageID,
				str,
			)
			if len(markup.InlineKeyboard) != 0 {
				msg.ReplyMarkup = &markup
			}
			msg.ParseMode = tgbotapi.ModeHTML
			if _, err := bot.TG.Request(msg); err != nil {
				return nilMsg, err
			}

			return nilMsg, nil
		} else {
			// Фото было, но теперь его не будет
			del := tgbotapi.NewDeleteMessage(
				editMsg[0].Chat.ID,
				editMsg[0].MessageID,
			)
			if _, err := bot.TG.Request(del); err != nil {
				return nilMsg, err
			}

			msg := tgbotapi.NewMessage(id, str)
			if len(markup.InlineKeyboard) != 0 {
				msg.ReplyMarkup = &markup
			}
			msg.ParseMode = tgbotapi.ModeHTML

			return bot.TG.Send(msg)
		}
	} else {
		// Обновлений нет, новое сообщение
		if imageID != "" {
			// С фото
			newMsg := tgbotapi.NewPhoto(
				id,
				tgbotapi.FileID(imageID),
			)
			newMsg.Caption = str
			newMsg.ParseMode = tgbotapi.ModeHTML
			if len(markup.InlineKeyboard) != 0 {
				newMsg.ReplyMarkup = &markup
			}

			return bot.TG.Send(newMsg)
		}
		// Только текст
		msg := tgbotapi.NewMessage(id, str)
		if len(markup.InlineKeyboard) != 0 {
			msg.ReplyMarkup = &markup
		} else {
			msg.ReplyMarkup = GeneralKeyboard(false)
		}
		msg.ParseMode = tgbotapi.ModeHTML

		return bot.TG.Send(msg)

	}
}

// Расшифровывать содержимое кнопки из карточки с расписанием
func ParseQuery(data []string) (SummaryType, database.ShedulesInUser, int, error) {
	var shedule database.ShedulesInUser
	isGroup := data[2] == "group"
	sheduleID, err := strconv.ParseInt(data[4], 0, 64)
	if err != nil {
		return Near, shedule, 0, err
	}
	shedule.IsGroup = isGroup
	shedule.SheduleId = sheduleID
	dt, err := strconv.ParseInt(data[3], 0, 0)
	if err != nil {
		return Near, shedule, 0, err
	}
	var sumType SummaryType
	switch data[1] {
	case "day":
		sumType = Day
	case "week":
		sumType = Week
	case "ics":
		sumType = ICS
	default:
		sumType = Near
	}

	return sumType, shedule, int(dt), nil
}

var SumKey = []string{"near", "day", "week"}

// Проверить строку на наличие одного из ключевых слов
func KeywordContains(str string, keywords []string) bool {
	for _, key := range keywords {
		if strings.Contains(str, key) {
			return true
		}
	}

	return false
}

// Меняем шило на мыло
func Swap(sh parser.WeekShedule) database.ShedulesInUser {
	return database.ShedulesInUser{
		IsGroup:   sh.IsGroup,
		SheduleId: sh.SheduleID,
	}
}

var Month = []string{
	"января",
	"февраля",
	"марта",
	"апреля",
	"мая",
	"июня",
	"июля",
	"августа",
	"сентября",
	"октября",
	"ноября",
	"декабря",
}
var weekdays = []string{
	"воскресенье",
	"понедельник",
	"вторник",
	"среду",
	"четверг",
	"пятницу",
	"субботу",
}
