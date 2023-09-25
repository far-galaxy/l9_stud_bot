package tg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssauparser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/builder"
)

var nilMsg = tgbotapi.Message{}

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
			"Просто напиши мне <b>номер группы</b> или <b>фамилию преподавателя</b>\n\n"+
			"Также можно получать уведомления о своих занятиях по кнопке <b>Моё расписание</b>👇\n\n"+
			"‼ Внимание! Бот ещё находится на стадии испытаний, поэтому могут возникать ошибки в его работе.\n"+
			"Рекомендуется сверять настоящее расписание и обо всех ошибках сообщать в чат"+
			"@chat_l9_stud_bot или по контактам в /help",
		GeneralKeyboard(false),
	)
}

// Поиск расписания по запросу
func (bot *Bot) Find(now time.Time, user *database.TgUser, query string) (tgbotapi.Message, error) {
	// Поиск в БД
	var groups []database.Group
	if err := bot.DB.Where(builder.Like{"GroupName", query}).Find(&groups); err != nil {
		return nilMsg, err
	}

	var teachers []database.Teacher
	if err := bot.DB.Where(builder.Like{"FirstName", query}).Find(&teachers); err != nil {
		return nilMsg, err
	}

	// Поиск на сайте
	list, siteErr := ssauparser.SearchInRasp(query)

	// Добавляем результаты поиска на сайте к результатам из БД
	allGroups, allTeachers := AppendSearchResults(list, groups, teachers)

	// Если получен единственный результат, сразу выдать (подключить) расписание
	if len(allGroups) == 1 || len(allTeachers) == 1 {
		var sheduleID int64
		var isGroup bool
		if len(allGroups) == 1 {
			sheduleID = allGroups[0].GroupId
			isGroup = true
		} else {
			sheduleID = allTeachers[0].TeacherId
			isGroup = false
		}
		shedule := ssauparser.WeekShedule{
			IsGroup:   isGroup,
			SheduleID: sheduleID,
		}
		notExists, _ := ssauparser.CheckGroupOrTeacher(bot.DB, shedule)

		return bot.ReturnSummary(notExists, user.PosTag == database.Add, user, shedule, now)

		// Если получено несколько групп
	} else if len(allGroups) != 0 {
		return bot.SendMsg(
			user,
			"Вот что я нашёл\nВыбери нужную группу",
			GenerateKeyboard(GenerateGroupsArray(allGroups, user.PosTag == database.Add)),
		)
		// Если получено несколько преподавателей
	} else if len(allTeachers) != 0 {
		return bot.SendMsg(
			user,
			"Вот что я нашёл\nВыбери нужного преподавателя",
			GenerateKeyboard(GenerateTeachersArray(allTeachers, user.PosTag == database.Add)),
		)
		// Если ничего не получено
	} else {
		var txt string
		if siteErr != nil {
			bot.Debug.Printf("sasau error: %s", siteErr)
			txt = "К сожалению, у меня ничего не нашлось, а на сайте ssau.ru/rasp произошла какая-то ошибка :(\n" +
				"Повтори попытку позже"
		} else {
			txt = "К сожалению, я ничего не нашёл ):\nПроверь свой запрос"
		}

		return bot.SendMsg(
			user,
			txt,
			GeneralKeyboard(false),
		)
	}
}

func AppendSearchResults(
	list ssauparser.SearchResults,
	groups []database.Group,
	teachers []database.Teacher,
) (
	[]database.Group,
	[]database.Teacher,
) {
	allGroups := groups
	allTeachers := teachers
	for _, elem := range list {
		if strings.Contains(elem.URL, "group") {
			exists := false
			for _, group := range groups {
				if elem.ID == group.GroupId {
					exists = true

					break
				}
			}
			if !exists {
				allGroups = append(allGroups, database.Group{GroupId: elem.ID, GroupName: elem.Text})
			}
		}
		if strings.Contains(elem.URL, "staff") {
			exists := false
			for _, teacher := range teachers {
				if elem.ID == teacher.TeacherId {
					exists = true

					break
				}
			}
			if !exists {
				teacher := ssauparser.ParseTeacherName(elem.Text)
				teacher.TeacherId = elem.ID
				allTeachers = append(allTeachers, teacher)
			}
		}
	}

	return allGroups, allTeachers
}

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
				GeneralKeyboard(false),
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
			GeneralKeyboard(true),
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
		err = bot.CreateICS(now[0], user, shedule, isPersonal, dt, *query)
	default:
		_, err = bot.GetShortSummary(now[0], user, shedule, isPersonal, *query.Message)
	}

	return err
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

		return bot.SendMsg(user, "Группа отключена", GeneralKeyboard(false))
	}

	return bot.SendMsg(user, "Действие отменено", GeneralKeyboard(true))
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
			"Ой, установлено слишком малое время. Попробуй ввести большее время",
			CancelKey(),
		)
	} else if t > 240 {
		return bot.SendMsg(
			user,
			"Ой, установлено слишком большое время. Попробуй ввести меньшее время",
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

	return bot.SendMsg(user, "Время установлено", GeneralKeyboard(true))
}
