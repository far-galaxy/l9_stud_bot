package tg

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	td "github.com/mergestat/timediff"
	"xorm.io/xorm"
)

/*
func (bot *Bot) GetPersonalSummary(user *database.TgUser, msg ...tgbotapi.Message) {
	var shedules []database.ShedulesInUser
	bot.DB.ID(user.L9Id).Find(&shedules)

	if len(shedules) == 0 {
		bot.Etc(user)
		return
	} else {
		err := bot.GetSummary(msg.Time(), user, shedules, true, msg...)
		if err != nil {
			log.Fatal(err)
		}
	}
}*/

func (bot *Bot) GetPersonal(now time.Time, user *database.TgUser, editMsg ...tgbotapi.Message) (tgbotapi.Message, error) {
	var shedules []database.ShedulesInUser
	bot.DB.ID(user.L9Id).Find(&shedules)

	if len(shedules) == 0 {
		user.PosTag = database.Add
		if _, err := bot.DB.ID(user.L9Id).Update(user); err != nil {
			return tgbotapi.Message{}, err
		}

		msg := tgbotapi.NewMessage(
			user.TgId,
			"У тебя пока никакого расписания не подключено\n"+
				"Введи <b>номер группы</b> или <b>фамилию преподавателя</b>",
		)
		msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{RemoveKeyboard: true}
		msg.ParseMode = tgbotapi.ModeHTML
		return bot.TG.Send(msg)
	} else {
		return bot.GetSummary(now, user, shedules, true, editMsg...)
	}
}

// Получить краткую сводку
func (bot *Bot) GetSummary(
	now time.Time,
	user *database.TgUser,
	shedules []database.ShedulesInUser,
	isPersonal bool,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {

	nilMsg := tgbotapi.Message{}
	lessons, err := bot.GetLessons(shedules, now)
	if err != nil {
		return nilMsg, err
	}
	if len(lessons) != 0 {
		var firstPair, secondPair []database.Lesson
		pairs := GroupPairs(lessons)
		firstPair = pairs[0]
		log.Println(firstPair, secondPair)
		str := "📝Краткая сводка:\n\n"
		if pairs[0][0].Begin.Day() != now.Day() {
			str += "❗️Сегодня пар нет\nБлижайшие занятия "
			str += td.TimeDiff(
				firstPair[0].Begin,
				td.WithLocale("ru_RU"),
				td.WithStartTime(now),
			)
			if firstPair[0].Begin.Sub(now).Hours() > 36 {
				str += fmt.Sprintf(
					", <b>%d %s</b>",
					firstPair[0].Begin.Day(),
					month[firstPair[0].Begin.Month()-1],
				)
			}
			str += "\n\n"
			day, err := bot.StrDayShedule(pairs, shedules[0].IsGroup)
			if err != nil {
				return nilMsg, err
			}
			str += day
		} else {
			if firstPair[0].Begin.Before(now) {
				str += "Сейчас:\n\n"
			} else {
				dt := td.TimeDiff(
					firstPair[0].Begin,
					td.WithLocale("ru_RU"),
					td.WithStartTime(now),
				)
				str += fmt.Sprintf("Ближайшая пара %s:\n\n", dt)
			}
			firstStr, err := PairToStr(firstPair, bot.DB, shedules[0].IsGroup)
			if err != nil {
				return nilMsg, err
			}
			str += firstStr
			if len(pairs) > 1 {
				secondPair = pairs[1]
				if firstPair[0].Begin.Day() == secondPair[0].Begin.Day() {
					str += "\nПосле неё:\n\n"
					secondStr, err := PairToStr(secondPair, bot.DB, shedules[0].IsGroup)
					if err != nil {
						return nilMsg, err
					}
					str += secondStr
				} else {
					str += "\nБольше ничего сегодня нет"
				}
			} else {
				str += "\nБольше ничего сегодня нет"
			}

		}

		var shId int64
		if isPersonal {
			shId = 0
		} else {
			shId = shedules[0].SheduleId
		}

		markup := SummaryKeyboard(
			// TODO: создать тип таких префиксов
			"sh_near",
			shId,
			shedules[0].IsGroup,
			0,
		)
		return bot.EditOrSend(user.TgId, str, "", markup, editMsg...)

	} else {
		msg := tgbotapi.NewMessage(user.TgId, "Ой! Занятий не обнаружено ):")
		return bot.TG.Send(msg)
	}
}

// ПОлучить расписание на день
func (bot *Bot) GetDaySummary(
	now time.Time,
	user *database.TgUser,
	shedules []database.ShedulesInUser,
	dt int,
	isPersonal bool,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	nilMsg := tgbotapi.Message{}
	day := time.Date(now.Year(), now.Month(), now.Day()+dt, 0, 0, 0, 0, now.Location())
	lessons, err := bot.GetLessons(shedules, day)
	if err != nil {
		return nilMsg, err
	}
	if len(lessons) != 0 {
		pairs := GroupPairs(lessons)
		var str string
		firstPair := pairs[0][0].Begin
		dayStr := DayStr(day)

		var shId int64
		if isPersonal {
			shId = 0
		} else {
			shId = shedules[0].SheduleId
		}
		markup := SummaryKeyboard(
			"sh_day",
			shId,
			shedules[0].IsGroup,
			dt,
		)

		if firstPair.Day() != day.Day() {
			str = fmt.Sprintf("В %s, занятий нет", dayStr)
			return bot.EditOrSend(user.TgId, str, "", markup, editMsg...)
		}
		str = fmt.Sprintf("Расписание на %s\n\n", dayStr)

		// TODO: придумать скачки для пустых дней
		//dt += int(firstPair.Sub(day).Hours()) / 24
		day, err := bot.StrDayShedule(pairs, shedules[0].IsGroup)
		if err != nil {
			return nilMsg, err
		}
		str += day
		return bot.EditOrSend(user.TgId, str, "", markup, editMsg...)
	} else {
		msg := tgbotapi.NewMessage(user.TgId, "Ой! Пар не обнаружено ):")
		return bot.TG.Send(msg)
	}

}

// Строка даты формата "среду, 1 января"
func DayStr(day time.Time) string {
	dayStr := fmt.Sprintf(
		"%s, <b>%d %s</b>",
		weekdays[int(day.Weekday())],
		day.Day(),
		month[day.Month()-1],
	)
	return dayStr
}

// Получить список ближайших занятий (для краткой сводки или расписания на день)
func (bot *Bot) GetLessons(shedules []database.ShedulesInUser, now time.Time) ([]database.Lesson, error) {

	condition := CreateCondition(shedules)

	var lessons []database.Lesson
	err := bot.DB.
		Where("end > ?", now.Format("2006-01-02 15:04:05")).
		And(condition).
		OrderBy("begin").
		Limit(32).
		Find(&lessons)

	return lessons, err
}

// Загрузка расписания из ssau.ru/rasp
func (bot *Bot) LoadShedule(shedule ssau_parser.WeekShedule) error {
	sh := ssau_parser.WeekShedule{
		SheduleId: shedule.SheduleId,
		IsGroup:   shedule.IsGroup,
	}
	for week := 1; week < 21; week++ {
		sh.Week = week
		err := sh.DownloadById(true)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				break
			}
			return err
		}
		_, _, err = ssau_parser.UpdateSchedule(bot.DB, sh)
		if err != nil {
			return err
		}
	}

	return nil
}

// Создать условие поиска группы/преподавателя
func CreateCondition(shedules []database.ShedulesInUser) string {
	var groups []string
	var teachers []string

	for _, sh := range shedules {
		if !sh.IsGroup {
			teachers = append(teachers, strconv.FormatInt(sh.SheduleId, 10))
		} else {
			groups = append(groups, strconv.FormatInt(sh.SheduleId, 10))
		}
	}

	var condition, teachers_str, groups_str string
	if len(groups) > 0 {
		groups_str = strings.Join(groups, ",")
		condition = "groupId in (" + groups_str + ") "
	}
	if len(teachers) > 0 {
		if len(condition) > 0 {
			condition += " or "
		}
		teachers_str += strings.Join(teachers, ",")
		condition += "teacherId in (" + teachers_str + ") "
	}
	return condition
}

// Группировка занятий по парам
func GroupPairs(lessons []database.Lesson) [][]database.Lesson {
	var shedule [][]database.Lesson
	var pair []database.Lesson

	l_idx := 0

	for l_idx < len(lessons) {
		day := lessons[l_idx].Begin
		for l_idx < len(lessons) && lessons[l_idx].Begin == day {
			pair = append(pair, lessons[l_idx])
			l_idx++
		}
		shedule = append(shedule, pair)
		pair = []database.Lesson{}
	}
	return shedule
}

// Конвертация занятий с текст
func PairToStr(pair []database.Lesson, db *xorm.Engine, isGroup bool) (string, error) {
	var str string
	beginStr := pair[0].Begin.Format("15:04")
	var endStr string
	if pair[0].Type == "mil" {
		endStr = "∞"
	} else {
		endStr = pair[0].End.Format("15:04")
	}
	str = fmt.Sprintf("📆 %s - %s\n", beginStr, endStr)

	var groups []database.Lesson
	if !isGroup {
		groups = pair[:]
		pair = pair[:1]
	}

	for i, sublesson := range pair {
		var type_emoji string
		switch sublesson.Type {
		case "lect":
			type_emoji = "📗"
		case "pract":
			type_emoji = "📕"
		case "lab":
			type_emoji = "📘"
		case "other":
			type_emoji = "📙"
		case "mil":
			type_emoji = "🫡"
		case "window":
			type_emoji = "🏝"
		default:
			type_emoji = "📙"
		}
		str += fmt.Sprintf("%s%s\n", type_emoji, sublesson.Name)
		if sublesson.Place != "" {
			str += fmt.Sprintf("🧭 %s\n", sublesson.Place)
		}
		if !isGroup {
			break
		}
		if sublesson.TeacherId != 0 {
			var t database.Teacher
			_, err := db.ID(sublesson.TeacherId).Get(&t)
			if err != nil {
				return "", err
			}
			str += fmt.Sprintf("👤 %s %s\n", t.FirstName, t.ShortName)
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
			var t database.Group
			_, err := db.ID(gr.GroupId).Get(&t)
			if err != nil {
				return "", err
			}
			str += fmt.Sprintf("👥 %s\n", t.GroupName)
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
