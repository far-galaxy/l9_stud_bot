package tg

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"xorm.io/xorm"
)

func (bot *Bot) GetSummary() {
	now := time.Now()
	log.Println(now.Format("01-02-2006 15:04:05 -07"), now.Format("01-02-2006 15:04:05"))

	var lessons []database.Lesson
	var shedules []database.ShedulesInUser
	bot.DB.ID(bot.TG_user.L9Id).Find(&shedules)

	var groups []string
	var teachers []string

	for _, sh := range shedules {
		if sh.IsTeacher {
			teachers = append(teachers, strconv.FormatInt(sh.SheduleId, 10))
		} else {
			groups = append(groups, strconv.FormatInt(sh.SheduleId, 10))
		}
	}

	teachers_str := strings.Join(teachers, ",")
	groups_str := strings.Join(groups, ",")

	bot.DB.
		Where("begin > ?", now.Format("2006-01-02 15:04:05")).
		And("groupId in (?) or teacherId in (?)", groups_str, teachers_str).
		OrderBy("begin").
		Limit(6).
		Find(&lessons)

	log.Println(lessons)

	if len(lessons) != 0 {
		var firstPair, secondPair []database.Lesson
		l_idx := 0
		day := lessons[0].Begin
		// Я хз, надо ли упарываться для случаев с более чем двумя подпарами
		for lessons[l_idx].Begin == day && l_idx < len(lessons) {
			firstPair = append(firstPair, lessons[l_idx])
			l_idx++
		}
		if l_idx < len(lessons) {
			day = lessons[l_idx].Begin
			for lessons[l_idx].Begin == day && l_idx < len(lessons) {
				secondPair = append(secondPair, lessons[l_idx])
				l_idx++
			}
		}
		log.Println(firstPair, secondPair)

		var str string
		if firstPair[0].Begin.Day() != time.Now().Day() {
			str = "❗️Сегодня пар нет\nБлижайшие занятия "
			if time.Until(firstPair[0].Begin).Hours() < 48 {
				str += "завтра\n"
			} else {
				str += fmt.Sprintf("%s\n\n", firstPair[0].Begin.Format("02.01"))
			}
		}

		firstPairStr, _ := PairToStr(firstPair, &bot.DB)
		str += firstPairStr

		if len(secondPair) != 0 && firstPair[0].Begin.Day() == secondPair[0].Begin.Day() {
			secondPairStr, _ := PairToStr(secondPair, &bot.DB)
			str += secondPairStr
		}
		msg := tgbotapi.NewMessage(bot.TG_user.TgId, str)
		bot.TG.Send(msg)
	}

}

func PairToStr(pair []database.Lesson, db *xorm.Engine) (string, error) {
	var str string
	beginStr := pair[0].Begin.Format("15:04")
	endStr := pair[0].End.Format("15:04")
	str = fmt.Sprintf("📆 %s - %s\n", beginStr, endStr)

	for _, sublesson := range pair {
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
			str += fmt.Sprintf("🧭%s\n", sublesson.Place)
		}
		if sublesson.TeacherId != 0 {
			var t database.Teacher
			_, err := db.ID(sublesson.TeacherId).Get(&t)
			if err != nil {
				return "", err
			}
			name := fmt.Sprintf("%s %s.%s.", t.LastName, t.FirstName[0:2], t.MidName[0:2])
			str += fmt.Sprintf("👤%s\n", name)
		}
		if sublesson.SubGroup != "" {
			str += fmt.Sprintf("👥%s\n", sublesson.SubGroup)
		}
		if sublesson.Comment != "" {
			str += fmt.Sprintf("💬%s\n", sublesson.Comment)
		}
		str += "--------------------------------\n"
	}
	return str, nil
}
