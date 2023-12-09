package tg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
)

var nilMsg = tgbotapi.Message{}
var nilKey = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}

// Приветственное сообщение
func (bot *Bot) Start(user *database.TgUser) (tgbotapi.Message, error) {
	user.PosTag = database.Ready
	_, err := bot.DB.ID(user.L9Id).Update(user)
	if err != nil {
		return nilMsg, err
	}

	return bot.SendMsg(
		user,
		bot.StartTxt,
		nilKey,
	)
}

// Выдача расписания
func (bot *Bot) ReturnSummary(
	notExists bool,
	user *database.TgUser,
	shedule ssauparser.WeekShedule,
	now time.Time,
) (
	tgbotapi.Message,
	error,
) {
	if notExists {
		msg := tgbotapi.NewMessage(user.TgId, "Загружаю расписание...\nЭто займёт некоторое время")
		Smsg, _ := bot.TG.Send(msg)
		_, _, err := bot.LoadShedule(shedule, now, false)
		if err != nil {
			return nilMsg, err
		}
		del := tgbotapi.NewDeleteMessage(Smsg.Chat.ID, Smsg.MessageID)
		if _, err := bot.TG.Request(del); err != nil {
			return nilMsg, err
		}
	}

	userSchedule := database.Schedule{
		TgUser:     user,
		IsPersonal: false,
		IsGroup:    shedule.IsGroup,
		ScheduleID: shedule.SheduleID,
	}
	if _, err := bot.ActShedule(&userSchedule); err != nil {
		return nilMsg, err
	}

	return nilMsg, bot.GetWeekSummary(now, userSchedule, -1, "")

}

// Получить расписание из кнопки
func (bot *Bot) GetShedule(user *database.TgUser, query *tgbotapi.CallbackQuery, now ...time.Time) error {
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	data := strings.Split(query.Data, "_")
	if len(data) != 3 {
		return fmt.Errorf("wrong button format: %s", query.Data)
	}
	isGroup := data[1] == Group
	groupID, err := strconv.ParseInt(data[2], 0, 64)
	if err != nil {
		return err
	}
	shedule := ssauparser.WeekShedule{
		IsGroup:   isGroup,
		SheduleID: groupID,
	}
	notExists, _ := ssauparser.CheckGroupOrTeacher(bot.DB, shedule)
	del := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
	if _, err := bot.TG.Request(del); err != nil {
		return err
	}
	_, err = bot.ReturnSummary(notExists, user, shedule, now[0])

	return err
}

// Обработка нажатия кнопки в карточке с расписанием
func (bot *Bot) HandleSummary(user *database.TgUser, query *tgbotapi.CallbackQuery, now ...time.Time) error {
	data := strings.Split(query.Data, "_")
	sumType, shedule, dt, err := ParseQuery(data)
	shedule.TgUser = user
	if err != nil {
		return err
	}
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	switch sumType {
	case Day:
		_, err = bot.GetDaySummary(now[0], shedule, dt, *query.Message)
	case Week:
		err = bot.GetWeekSummary(now[0], shedule, dt, "", *query.Message)
	case ICS:
		err = bot.CreateICS(shedule, *query)
	case Connect:
		_, err = bot.ConnectShedule(shedule, *query.Message)
	case Session:
		_, err = bot.GetSession(shedule, *query.Message)
	}

	return err
}

// Подключение уведомлений
func (bot *Bot) ConnectShedule(
	sh database.Schedule,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	shedules := database.ShedulesInUser{
		L9Id: sh.TgUser.L9Id,
	}
	exists, err := bot.DB.Get(&shedules)
	if err != nil {
		return nilMsg, err
	}
	if exists {
		return bot.SendMsg(
			sh.TgUser,
			"У тебя уже подключено одно расписание!\n"+
				"Сначали отключи его в меню /options, затем можешь подключить другое",
			nilKey,
		)
	}

	if !sh.IsGroup {
		return bot.SendMsg(
			sh.TgUser,
			"Личное расписание пока не работает с преподавателями :(\n"+
				"Приносим извинения за временные неудобства",
			nilKey,
		)
	}
	shedules = database.ShedulesInUser{
		L9Id:      sh.TgUser.L9Id,
		IsGroup:   sh.IsGroup,
		SheduleId: sh.ScheduleID,
		FirstTime: 45,
		First:     true,
		NextNote:  true,
		NextDay:   true,
		NextWeek:  true,
	}
	if _, err := bot.DB.InsertOne(&shedules); err != nil {
		return nilMsg, err
	}
	sh.TgUser.PosTag = database.Ready
	if _, err := bot.DB.ID(sh.TgUser.L9Id).Update(sh.TgUser); err != nil {
		return nilMsg, err
	}

	return bot.EditOrSend(
		sh.TgUser.TgId,
		"Расписание успешно подключено!\n"+
			"Теперь можно смотреть свои занятия по команде <b>/schedule</b>\n\n"+
			"Также ты будешь получать уведомления о занятиях, "+
			"которыми можно управлять по команде <b>/options</b>\n",
		"",
		nilKey,
		editMsg[0],
	)
}

func (bot *Bot) Etc(user *database.TgUser) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(user.TgId, "Oй!")

	return bot.TG.Send(msg)
}

func (bot *Bot) Cancel(user *database.TgUser, query *tgbotapi.CallbackQuery) error {
	user.PosTag = database.Ready
	_, err := bot.DB.ID(user.L9Id).Update(user)
	if err != nil {
		return err
	}
	if query.ID != "" {
		callback := tgbotapi.NewCallback(query.ID, "Действие отменено")
		_, err = bot.TG.Request(callback)
		if err != nil {
			return err
		}
	}
	del := tgbotapi.NewDeleteMessage(query.From.ID, query.Message.MessageID)
	_, err = bot.TG.Request(del)

	return err
}

func (bot *Bot) DeleteGroup(user *database.TgUser, text string) (tgbotapi.Message, error) {
	user.PosTag = database.Ready
	if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
		return nilMsg, err
	}
	if strings.ToLower(text) == "да" {
		userInfo := database.ShedulesInUser{
			L9Id: user.L9Id,
		}
		if _, err := bot.DB.Delete(&userInfo); err != nil {
			return nilMsg, err
		}
		files := database.File{
			TgId:       user.TgId,
			IsPersonal: true,
		}
		if _, err := bot.DB.UseBool("IsPersonal").Delete(&files); err != nil {
			return nilMsg, err
		}

		return bot.SendMsg(user, "Группа отключена", nil)
	}

	return bot.SendMsg(user, "Действие отменено", nil)
}

func (bot *Bot) SetFirstTime(msg *tgbotapi.Message, user *database.TgUser) (tgbotapi.Message, error) {
	t, err := strconv.Atoi(msg.Text)
	if err != nil {
		return bot.SendMsg(
			user,
			"Ой, время соообщения о начале занятий введено как-то неверно ):",
			CancelKey(),
		)
	}
	userInfo := database.ShedulesInUser{
		L9Id: user.L9Id,
	}
	if _, err := bot.DB.Get(&userInfo); err != nil {
		return nilMsg, err
	}
	if t <= 10 {
		return bot.SendMsg(
			user,
			"Ой, установлено слишком малое время. Попробуй ввести большее время (не менее 15 минут)",
			CancelKey(),
		)
	} else if t > 240 {
		return bot.SendMsg(
			user,
			"Ой, установлено слишком большое время. Попробуй ввести меньшее время (не более 240 минут)",
			CancelKey(),
		)
	}
	userInfo.FirstTime = t / 5 * 5
	if _, err := bot.DB.ID(userInfo.UID).Update(userInfo); err != nil {
		return nilMsg, err
	}
	user.PosTag = database.Ready
	if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
		return nilMsg, err
	}

	return bot.SendMsg(user, "Время установлено", nil)
}
