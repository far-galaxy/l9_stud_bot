package tg

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
	"xorm.io/xorm"
)

// Получение расписания из команды /{group, staff} ID_ расписания
func (bot *Bot) GetSheduleFromCmd(
	now time.Time,
	user *database.TgUser,
	query string,
) (
	tgbotapi.Message,
	error,
) {
	isGroup := strings.Contains(query, "/group")
	cmd := strings.Split(query, " ")
	if len(cmd) == 1 {
		return bot.SendMsg(user, "Необходимо указать ID расписания",
			nilKey)
	}
	sheduleID, err := strconv.ParseInt(cmd[1], 10, 64)
	if err != nil {
		return bot.SendMsg(user, "Некорректный ID расписания",
			nilKey)
	}
	/*
		shedule := ssauparser.WeekShedule{
			IsGroup:   isGroup,
			SheduleID: sheduleID,
		}
	*/
	shedule := database.Schedule{
		TgUser:     user,
		IsGroup:    isGroup,
		ScheduleID: sheduleID,
	}
	//notExists, _ := ssauparser.CheckGroupOrTeacher(bot.DB, shedule)

	//return bot.ReturnSummary(notExists, user, shedule, now)
	return bot.GetSession(shedule)
}

// Создания сообщения с расписанием сессии
func (bot *Bot) GetSession(
	shedule database.Schedule,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	if _, err := bot.ActShedule(&shedule); err != nil {
		return nilMsg, err
	}
	query := "GroupId = ?"
	if !shedule.IsGroup {
		query = "TeacherId = ?"
	}
	var lessons []database.Lesson
	if err := bot.DB.
		In("Type", database.Consult, database.Exam).
		Where(query, shedule.ScheduleID).
		Asc("Begin").
		Find(&lessons); err != nil {
		return nilMsg, err
	}
	str := "<b>Расписание сессии:</b>\n\n"
	if len(lessons) == 0 {
		str = "Расписания сессии тут пока нет\n"
		if shedule.IsPersonal {
			str += "Как только оно появится, я обязательно сообщу!"
		}

		markup := SummaryKeyboard(
			Week,
			shedule,
			0,
			false,
		)

		return bot.EditOrSend(shedule.TgUser.TgId, str, "", markup, editMsg...)
	}

	for i, l := range lessons {
		if i > 0 &&
			lessons[i-1].Name == l.Name &&
			lessons[i-1].Type == l.Type {
			continue
		}
		obj := fmt.Sprintf(
			l.Begin.Format("📆 <b>02 %s (%s) 15:04</b>\n"),
			Month[l.Begin.Month()-1],
			weekdaysNom[int(l.Begin.Weekday())],
		)
		obj += fmt.Sprintf("<i>%s</i>\n%s %s\n",
			l.Name, Icons[l.Type], Comm[l.Type],
		)
		if l.Place != "" {
			obj += fmt.Sprintf("🧭 %s\n", l.Place)
		}
		if l.TeacherId != 0 {
			staff, err := api.GetStaff(bot.DB, l.TeacherId)
			if err != nil {
				return nilMsg, err
			}
			obj += fmt.Sprintf("👤 %s %s\n", staff.FirstName, staff.ShortName)
		}
		if !shedule.IsGroup {
			group, err := api.GetGroup(bot.DB, l.GroupId)
			if err != nil {
				return nilMsg, err
			}
			obj += fmt.Sprintf("👥 %s\n", group.GroupName)
		}
		obj += "------------------------------------------\n"

		str += obj
	}

	connectButton := !shedule.IsPersonal && !bot.IsThereUserShedule(shedule.TgUser)
	markup := SummaryKeyboard(
		Session,
		shedule,
		0,
		connectButton,
	)

	return bot.SendMsg(shedule.TgUser, str, markup)
}

func (bot *Bot) GetPersonal(
	now time.Time,
	shedule database.Schedule,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	exists, err := bot.ActShedule(&shedule)
	if err != nil {
		return nilMsg, err
	}

	if !exists {
		return bot.SendMsg(
			shedule.TgUser,
			"У тебя пока никакого расписания не подключено\n\n"+
				"Введи <b>номер группы</b> "+
				"(в формате 2305 или 2305-240502D), "+
				"и в появившемся расписании нажми <b>🔔 Подключить уведомления</b>\n\n"+
				"https://youtube.com/shorts/FHE2YAGYBa8",
			tgbotapi.ReplyKeyboardRemove{RemoveKeyboard: true},
		)
	}

	//return bot.GetWeekSummary(now, shedule, -1, "", editMsg...)
	return bot.GetSession(shedule, editMsg...)

}

// Актуализация запроса на расписание для персональных расписаний
func (bot *Bot) ActShedule(schedule *database.Schedule) (bool, error) {
	var sh database.ShedulesInUser
	var exists bool
	var err error
	if schedule.IsPersonal {
		exists, err = bot.DB.Where("L9Id = ?", schedule.TgUser.L9Id).Get(&sh)
		if err != nil {
			return false, err
		}
		schedule.IsGroup = sh.IsGroup
		schedule.ScheduleID = sh.SheduleId
	}

	return exists, nil
}

// Получить расписание на день
func (bot *Bot) GetDaySummary(
	now time.Time,
	schedule database.Schedule,
	dt int,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day = day.AddDate(0, 0, dt)
	if _, err := bot.ActShedule(&schedule); err != nil {
		return nilMsg, err
	}
	lessons, err := api.GetDayLessons(bot.DB, schedule, day)
	if err != nil {
		return nilMsg, err
	}
	dayStr := DayStr(day)
	connectButton := !schedule.IsPersonal && !bot.IsThereUserShedule(schedule.TgUser)
	markup := SummaryKeyboard(Day, schedule, dt, connectButton)

	if len(lessons) != 0 {
		pairs := api.GroupPairs(lessons)
		var str string
		firstPair := pairs[0][0].Begin

		if firstPair.Day() != day.Day() {
			str = fmt.Sprintf("В %s, занятий нет", dayStr)

			return bot.EditOrSend(schedule.TgUser.TgId, str, "", markup, editMsg...)
		}
		str = fmt.Sprintf("Расписание на %s\n\n", dayStr)

		// TODO: придумать скачки для пустых дней
		day, err := bot.StrDayShedule(pairs, schedule.IsGroup)
		if err != nil {
			return nilMsg, err
		}
		str += day

		return bot.EditOrSend(schedule.TgUser.TgId, str, "", markup, editMsg...)
	}
	str := fmt.Sprintf("В %s, занятий нет", dayStr)

	return bot.EditOrSend(schedule.TgUser.TgId, str, "", markup, editMsg...)
	//return bot.SendMsg(schedule.TgUser, "Ой! Пар не обнаружено ):", nil)
}

// Строка даты формата "среду, 1 января"
func DayStr(day time.Time) string {
	dayStr := fmt.Sprintf(
		"%s, <b>%d %s</b>",
		weekdays[int(day.Weekday())],
		day.Day(),
		Month[day.Month()-1],
	)

	return dayStr
}

// Загрузка расписания из ssau.ru/rasp
func (bot *Bot) LoadShedule(shedule ssauparser.WeekShedule, now time.Time, fast bool) (
	[]database.Lesson,
	[]database.Lesson,
	error,
) {
	sh := ssauparser.WeekShedule{
		SheduleID: shedule.SheduleID,
		IsGroup:   shedule.IsGroup,
	}
	var start, end int
	if fast {
		_, start = now.ISOWeek()
		start -= bot.Week
		end = start + 1
	} else {
		start = 1
		end = 25
	}
	var add, del []database.Lesson
	for week := start; week < end; week++ {
		sh.Week = week
		if err := sh.DownloadByID(true); err != nil {
			if strings.Contains(err.Error(), "404") {
				break
			}

			return nil, nil, err
		}
		a, d, err := ssauparser.UpdateSchedule(bot.DB, sh)
		if err != nil {
			return nil, nil, err
		}
		add = append(add, a...)
		del = append(del, d...)
	}
	// Обновляем время обновления
	if len(add) > 0 || len(del) > 0 {
		if sh.IsGroup {
			group, err := api.GetGroup(bot.DB, sh.SheduleID)
			if err != nil {
				return nil, nil, err
			}
			group.LastUpd = now
			if err := api.UpdateGroup(bot.DB, group); err != nil {
				return nil, nil, err
			}
		} else {
			staff, err := api.GetStaff(bot.DB, sh.SheduleID)
			if err != nil {
				return nil, nil, err
			}
			staff.LastUpd = now
			if err := api.UpdateStaff(bot.DB, staff); err != nil {
				return nil, nil, err
			}
		}
	}

	return add, del, nil
}

var Icons = map[database.Kind]string{
	database.Lection:    "📗",
	database.Practice:   "📕",
	database.Lab:        "📘",
	database.Other:      "📙",
	database.Military:   "🫡",
	database.Window:     "🏝",
	database.Exam:       "💀",
	database.Consult:    "🗨",
	database.CourseWork: "🤯",
	database.Test:       "📝",
	database.Unknown:    "❓",
}

var Comm = map[database.Kind]string{
	database.Lection:    "Лекция",
	database.Practice:   "Практика",
	database.Lab:        "Лаба",
	database.Other:      "Прочее",
	database.Military:   "",
	database.Window:     "",
	database.Exam:       "Экзамен",
	database.Consult:    "Консультация",
	database.CourseWork: "Курсовая",
	database.Test:       "Зачёт",
	database.Unknown:    "",
}

// Конвертация занятий с текст
func PairToStr(pair []database.Lesson, db *xorm.Engine, isGroup bool) (string, error) {
	var str string
	if len(pair) == 0 {
		return "", fmt.Errorf("empty pair")
	}
	beginStr := pair[0].Begin.Format("15:04")
	endStr := pair[0].End.Format("15:04")
	if pair[0].Type == database.Military {
		endStr = "∞"
	}
	str = fmt.Sprintf("📆 %s - %s\n", beginStr, endStr)

	var groups []database.Lesson
	if !isGroup {
		groups = pair[:]
		pair = pair[:1]
	}

	for i, sublesson := range pair {
		typeEmoji := Icons[sublesson.Type] + " " + Comm[sublesson.Type]
		str += fmt.Sprintf("%s %s\n", typeEmoji, sublesson.Name)
		if sublesson.Place != "" {
			str += fmt.Sprintf("🧭 %s\n", sublesson.Place)
		}
		if !isGroup {
			break
		}
		if sublesson.TeacherId != 0 {
			staff, err := api.GetStaff(db, sublesson.TeacherId)
			if err != nil {
				return "", err
			}
			str += fmt.Sprintf("👤 %s %s\n", staff.FirstName, staff.ShortName)
		}
		if sublesson.SubGroup != 0 {
			str += fmt.Sprintf("👥 Подгруппа: %d\n", sublesson.SubGroup)
		}
		if sublesson.Comment != "" {
			str += fmt.Sprintf("💬 %s\n", sublesson.Comment)
		}
		if i != len(pair)-1 {
			str += "+\n"
		}
	}

	if !isGroup {
		for _, gr := range groups {
			group, err := api.GetGroup(db, gr.GroupId)
			if err != nil {
				return "", err
			}
			str += fmt.Sprintf("👥 %s\n", group.GroupName)
			if gr.SubGroup != 0 {
				str += fmt.Sprintf("👥 Подгруппа: %d\n", gr.SubGroup)
			}
		}
		if pair[0].Comment != "" {
			str += fmt.Sprintf("💬 %s\n", pair[0].Comment)
		}
	}

	str += "------------------------------------------\n"

	return str, nil
}

// Текст расписания на день
func (bot *Bot) StrDayShedule(lessons [][]database.Lesson, isGroup bool) (string, error) {
	var str string
	day := lessons[0][0].Begin.Day()
	for _, pair := range lessons {
		if pair[0].Begin.Day() == day {
			line, err := PairToStr(pair, bot.DB, isGroup)
			if err != nil {
				return "", err
			}
			str += line
		} else {
			break
		}
	}

	return str, nil
}
