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

func (bot *Bot) GetPersonalSummary() {
	var shedules []database.ShedulesInUser
	bot.DB.ID(bot.TG_user.L9Id).Find(&shedules)

	if len(shedules) == 0 {
		bot.Etc()
		return
	} else {
		err := bot.GetSummary(shedules)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (bot *Bot) GetSummary(shedules []database.ShedulesInUser, isRetry ...bool) error {
	now := time.Now().Add(time.Hour * time.Duration(5) * (-1))
	log.Println(now.Format("01-02-2006 15:04:05 -07"), now.Format("01-02-2006 15:04:05"))

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

	var lessons []database.Lesson
	err := bot.DB.
		Where("end > ?", now.Format("2006-01-02 15:04:05")).
		And(condition).
		OrderBy("begin").
		Limit(16).
		Find(&lessons)

	if err != nil {
		return err
	}

	if len(lessons) != 0 {
		var firstPair, secondPair []database.Lesson
		pairs := GroupPairs(lessons)
		firstPair = pairs[0]
		secondPair = pairs[1]
		log.Println(firstPair, secondPair)

		var str string
		if pairs[0][0].Begin.Day() != time.Now().Day() {
			str = "❗️Сегодня пар нет\nБлижайшие занятия "
			if time.Until(firstPair[0].Begin).Hours() < 48 {
				str += "завтра\n"
			} else {
				str += fmt.Sprintf("%s\n\n", firstPair[0].Begin.Format("02.01"))
			}
			day, _ := bot.GetDayShedule(pairs)
			str += day
		} else {
			str = "Сводка на сегодня\n\n"
			day, _ := bot.GetDayShedule(pairs)
			str += day
		}

		msg := tgbotapi.NewMessage(bot.TG_user.TgId, str)
		bot.TG.Send(msg)
	} else if len(isRetry) == 0 {
		_, week := time.Now().ISOWeek()
		week -= bot.Week
		for _, sh := range shedules {
			doc, err := ssau_parser.ConnectById(sh.SheduleId, sh.IsTeacher, week)
			if err != nil {
				return err
			}
			shedule, err := ssau_parser.Parse(doc, !sh.IsTeacher, sh.SheduleId, week)
			if err != nil {
				return err
			}
			err = ssau_parser.UploadShedule(bot.DB, *shedule)
			if err != nil {
				return err
			}
		}
		bot.GetSummary(shedules, true)
	}
	return nil
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
