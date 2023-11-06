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
		"Привет! У меня можно посмотреть в удобном формате <b>ближайшие пары</b>"+
			", расписание <b>по дням</b> и даже <b>по неделям</b>!\n"+
			"Просто напиши мне <b>номер группы</b> или <b>фамилию преподавателя</b>\n"+
			fmt.Sprintf("(чтобы более удобно искать своё расписание, напиши сначала @%s , ", bot.Name)+
			"затем уже нужный запрос)\n\n"+
			"Также можно получать уведомления о своих занятиях, нажав на кнопку "+
			"<b>🔔 Подключить уведомления</b> в появившемся расписании\n\n"+
			"https://youtube.com/shorts/FHE2YAGYBa8\n\n"+
			"‼ Внимание! Бот ещё находится на стадии испытаний, поэтому могут возникать ошибки в его работе.\n"+
			"Рекомендуется сверять настоящее расписание и обо всех ошибках сообщать в чат "+
			"@chat_l9_stud_bot или по контактам в /help",
		nilKey,
	)
}

// Выдача расписания
func (bot *Bot) ReturnSummary(
	notExists bool,
	isAdd bool,
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

	if isAdd {
		if !shedule.IsGroup {
			return bot.SendMsg(
				user,
				"Личное расписание пока не работает с преподавателями :(\n"+
					"Приносим извинения за временные неудобства",
				nilKey,
			)
		}
		// Групповые чаты
		if user.TgId < 0 {
			group := database.GroupChatInfo{
				ChatID:    user.TgId,
				IsGroup:   shedule.IsGroup,
				SheduleID: shedule.SheduleID,
			}
			if _, err := bot.DB.UseBool().Update(&group); err != nil {
				return nilMsg, err
			}

			return bot.SendMsg(
				user,
				"Расписание успешно подключено!\n"+
					"Теперь по команде /shedule@l9_stud_bot ты сможешь открыть расписание на текущую неделю",
				nilKey,
			)
		}
		sh := Swap(shedule)
		sh.L9Id = user.L9Id
		sh.FirstTime = 45
		sh.First = true
		sh.NextNote = true
		sh.NextDay = true
		sh.NextWeek = true
		if _, err := bot.DB.InsertOne(&sh); err != nil {
			return nilMsg, err
		}
		user.PosTag = database.Ready
		if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
			return nilMsg, err
		}

		return bot.SendMsg(
			user,
			"Расписание успешно подключено!\n"+
				"Теперь можно смотреть свои занятия по кнопке <b>Моё расписание</b>👇\n\n"+
				"Также ты будешь получать уведомления о занятиях, "+
				"которыми можно управлять в панели <b>Настройки</b>\n",
			nil,
		)
	}

	return nilMsg, bot.GetWeekSummary(now, user, Swap(shedule), -1, false, "")

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
	isGroup := data[1] == "group"
	isAdd := data[0] == "true"
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
	_, err = bot.ReturnSummary(notExists, isAdd, user, shedule, now[0])

	return err
}

// Обработка нажатия кнопки в карточке с расписанием
func (bot *Bot) HandleSummary(user *database.TgUser, query *tgbotapi.CallbackQuery, now ...time.Time) error {
	data := strings.Split(query.Data, "_")
	sumType, shedule, dt, err := ParseQuery(data)
	if err != nil {
		return err
	}
	if len(now) == 0 {
		now = append(now, time.Now())
	}
	isPersonal := data[2] == "personal"
	switch sumType {
	case Day:
		_, err = bot.GetDaySummary(now[0], user, shedule, dt, isPersonal, *query.Message)
	case Week:
		err = bot.GetWeekSummary(now[0], user, shedule, dt, isPersonal, "", *query.Message)
	case ICS:
		err = bot.CreateICS(user, shedule, isPersonal, *query)
	case Connect:
		_, err = bot.ConnectShedule(user, shedule, *query.Message)
	// TODO: задел, если никому не понравится пересылка
	//case Session:
	//_, err = bot.GetSession(user, shedule, isPersonal, *query.Message)
	default:
		_, err = bot.GetShortSummary(now[0], user, shedule, isPersonal, *query.Message)
	}

	return err
}

// Подключение уведомлений
func (bot *Bot) ConnectShedule(
	user *database.TgUser,
	sh database.ShedulesInUser,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	shedules := database.ShedulesInUser{
		L9Id: user.L9Id,
	}
	exists, err := bot.DB.Get(&shedules)
	if err != nil {
		return nilMsg, err
	}
	if exists {
		return bot.SendMsg(
			user,
			"У тебя уже подключено одно расписание!\n"+
				"Сначали отключи его в меню /options, затем можешь подключить другое",
			nilKey,
		)
	}

	if !sh.IsGroup {
		return bot.SendMsg(
			user,
			"Личное расписание пока не работает с преподавателями :(\n"+
				"Приносим извинения за временные неудобства",
			nilKey,
		)
	}
	sh.L9Id = user.L9Id
	sh.FirstTime = 45
	sh.First = true
	sh.NextNote = true
	sh.NextDay = true
	sh.NextWeek = true
	if _, err := bot.DB.InsertOne(&sh); err != nil {
		return nilMsg, err
	}
	user.PosTag = database.Ready
	if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
		return nilMsg, err
	}

	return bot.EditOrSend(
		user.TgId,
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
