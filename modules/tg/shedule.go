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
	"xorm.io/xorm"
)

func (bot *Bot) GetPersonalSummary(msg ...tgbotapi.Message) {
	var shedules []database.ShedulesInUser
	bot.DB.ID(bot.TG_user.L9Id).Find(&shedules)

	if len(shedules) == 0 {
		bot.Etc()
		return
	} else {
		err := bot.GetSummary(shedules, true, msg...)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (bot *Bot) GetSummary(shedules []database.ShedulesInUser, isPersonal bool, editMsg ...tgbotapi.Message) error {
	now := time.Now() //.Add(time.Hour * time.Duration(5) * (-1))

	lessons, err := bot.GetLessons(shedules, now)
	if err != nil {
		return err
	}
	if len(lessons) != 0 {
		var firstPair, secondPair []database.Lesson
		pairs := GroupPairs(lessons)
		firstPair = pairs[0]
		log.Println(firstPair, secondPair)

		str := "📝Краткая сводка:\n\n"
		if pairs[0][0].Begin.Day() != time.Now().Day() {
			str += "❗️Сегодня пар нет\nБлижайшие занятия "
			if time.Until(firstPair[0].Begin).Hours() < 48 {
				str += "завтра\n"
			} else {
				str += fmt.Sprintf("%s\n\n", firstPair[0].Begin.Format("02.01"))
			}
			day, err := bot.GetDayShedule(pairs)
			if err != nil {
				return err
			}
			str += day
		} else {
			if firstPair[0].Begin.Before(now) {
				str += "Сейчас:\n\n"
			} else {
				str += "Ближайшая пара сегодня:\n\n"
			}
			firstStr, err := PairToStr(firstPair, bot.DB)
			if err != nil {
				return err
			}
			str += firstStr
			if len(pairs) > 1 {
				secondPair = pairs[1]
				if firstPair[0].Begin.Day() == secondPair[0].Begin.Day() {
					str += "\nПосле неё:\n\n"
					secondStr, err := PairToStr(secondPair, bot.DB)
					if err != nil {
						return err
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
			"near",
			shId,
			shedules[0].IsTeacher,
			0,
		)
		bot.EditOrSend(str, markup, editMsg...)

	} else {
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Ой! Пар не обнаружено ):")
		bot.TG.Send(msg)
	}
	return nil
}

func (bot *Bot) GetPersonalDaySummary(dt int, msg ...tgbotapi.Message) {
	var shedules []database.ShedulesInUser
	bot.DB.ID(bot.TG_user.L9Id).Find(&shedules)

	if len(shedules) == 0 {
		bot.Etc()
		return
	} else {
		err := bot.GetDaySummary(shedules, dt, true, msg...)
		if err != nil {
			log.Fatal(err)
		}
	}
}

var ruWeekdays = []string{
	"понедельник",
	"вторник",
	"среду",
	"четверг",
	"пятницу",
	"субботу",
}

func (bot *Bot) GetDaySummary(shedules []database.ShedulesInUser, dt int, isPersonal bool, editMsg ...tgbotapi.Message) error {
	now := time.Now()
	day := time.Date(now.Year(), now.Month(), now.Day()+dt, 0, 0, 0, 0, now.Location())
	lessons, err := bot.GetLessons(shedules, day)
	if err != nil {
		return err
	}
	if len(lessons) != 0 {
		pairs := GroupPairs(lessons)
		var str string

		str = fmt.Sprintf(
			"Расписание на %s, %s\n\n",
			ruWeekdays[int(pairs[0][0].Begin.Weekday())-1],
			pairs[0][0].Begin.Format("02.01"),
		)
		day, err := bot.GetDayShedule(pairs)
		if err != nil {
			return err
		}
		str += day

		var shId int64
		if isPersonal {
			shId = 0
		} else {
			shId = shedules[0].SheduleId
		}
		markup := SummaryKeyboard(
			"day",
			shId,
			shedules[0].IsTeacher,
			dt,
		)
		bot.EditOrSend(str, markup, editMsg...)
	} else {
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, "Ой! Пар не обнаружено ):")
		bot.TG.Send(msg)
	}

	return nil
}

func (bot *Bot) GetLessons(shedules []database.ShedulesInUser, now time.Time, isRetry ...int) ([]database.Lesson, error) {

	condition := CreateCondition(shedules)

	var lessons []database.Lesson
	err := bot.DB.
		Where("end > ?", now.Format("2006-01-02 15:04:05")).
		And(condition).
		OrderBy("begin").
		Limit(16).
		Find(&lessons)

	if err != nil {
		return nil, err
	}

	if len(isRetry) == 0 || isRetry[0] < 2 {
		_, week := now.ISOWeek()
		isRetry, err = bot.LoadShedule(shedules, week, isRetry...)
		if err != nil {
			return nil, err
		}
		dw := isRetry[0]
		return bot.GetLessons(shedules, now, dw+1)
	} else if len(isRetry) != 0 && len(lessons) != 0 {
		return lessons, nil
	} else {
		return nil, nil
	}
}

func (bot *Bot) LoadShedule(shedules []database.ShedulesInUser, week int, isRetry ...int) ([]int, error) {
	if len(isRetry) == 0 {
		isRetry = []int{0}
	}
	dw := isRetry[0]
	week -= bot.Week
	for _, sh := range shedules {
		doc, err := ssau_parser.DownloadSheduleById(sh.SheduleId, sh.IsTeacher, week+dw)
		if err != nil {
			return nil, err
		}
		shedule, err := ssau_parser.Parse(doc, !sh.IsTeacher, sh.SheduleId, week+dw)
		if err != nil {
			return nil, err
		}
		err = ssau_parser.UploadShedule(bot.DB, *shedule)
		if err != nil {
			return nil, err
		}
	}
	return isRetry, nil
}

func CreateCondition(shedules []database.ShedulesInUser) string {
	var groups []string
	var teachers []string

	for _, sh := range shedules {
		if sh.IsTeacher {
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

func (bot *Bot) GetDayShedule(lessons [][]database.Lesson) (string, error) {
	var str string
	day := lessons[0][0].Begin.Day()
	for _, pair := range lessons {
		if pair[0].Begin.Day() == day {
			line, err := PairToStr(pair, bot.DB)
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

func PairToStr(pair []database.Lesson, db *xorm.Engine) (string, error) {
	var str string
	beginStr := pair[0].Begin.Format("15:04")
	endStr := pair[0].End.Format("15:04")
	str = fmt.Sprintf("📆 %s - %s\n", beginStr, endStr)

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
		default:
			type_emoji = "📙"
		}
		str += fmt.Sprintf("%s%s\n", type_emoji, sublesson.Name)
		if sublesson.Place != "" {
			str += fmt.Sprintf("🧭 %s\n", sublesson.Place)
		}
		if sublesson.TeacherId != 0 {
			var t database.Teacher
			_, err := db.ID(sublesson.TeacherId).Get(&t)
			if err != nil {
				return "", err
			}
			name := GenerateName(t)
			str += fmt.Sprintf("👤 %s\n", name)
		}
		if sublesson.SubGroup != "" {
			str += fmt.Sprintf("👥 %s\n", sublesson.SubGroup)
		}
		if sublesson.Comment != "" {
			str += fmt.Sprintf("💬 %s\n", sublesson.Comment)
		}
		if i != len(pair)-1 {
			str += "+\n"
		}
	}

	str += "------------------------------------------\n"
	return str, nil
}
