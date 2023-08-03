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

func (bot *Bot) GetPersonalSummary(user *database.TgUser, msg ...tgbotapi.Message) {
	var shedules []database.ShedulesInUser
	bot.DB.ID(user.L9Id).Find(&shedules)

	if len(shedules) == 0 {
		bot.Etc(user)
		return
	} else {
		err := bot.GetSummary(user, shedules, true, msg...)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Получить краткую сводку
func (bot *Bot) GetSummary(
	user *database.TgUser,
	shedules []database.ShedulesInUser,
	isPersonal bool,
	editMsg ...tgbotapi.Message) error {

	now, _ := time.Parse("2006-01-02 15:04 -07", "2023-03-06 07:20 +04") //time.Now().Add(time.Hour * time.Duration(24) * (-1) * 30 * 4)

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
		if pairs[0][0].Begin.Day() != now.Day() {
			str += "❗️Сегодня пар нет\nБлижайшие занятия "
			if firstPair[0].Begin.Sub(now).Hours() < 48 {
				str += "завтра\n"
			} else {
				// TODO: добавить прописные названия месяцев
				str += fmt.Sprintf("%s\n\n", firstPair[0].Begin.Format("02.01"))
			} /*
				day, err := bot.GetDayShedule(pairs)
				if err != nil {
					return err
				}
				str += day*/
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
		/*
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
			)*/
		bot.EditOrSend(user.TgId, str, tgbotapi.NewInlineKeyboardMarkup(), editMsg...)

	} else {
		msg := tgbotapi.NewMessage(user.TgId, "Ой! Пар не обнаружено ):")
		bot.TG.Send(msg)
	}
	return nil
}

// Получить список ближайших занятий (для краткой сводки или расписания на день)
func (bot *Bot) GetLessons(shedules []database.ShedulesInUser, now time.Time) ([]database.Lesson, error) {

	condition := CreateCondition(shedules)

	var lessons []database.Lesson
	err := bot.DB.
		Where("end > ?", now.Format("2006-01-02 15:04:05")).
		And(condition).
		OrderBy("begin").
		Limit(16).
		Find(&lessons)

	return lessons, err
}

// Загрузка расписания из ssau.ru/rasp
func (bot *Bot) LoadShedule(shedule ssau_parser.WeekShedule) error {
	sh := ssau_parser.WeekShedule{
		SheduleId: shedule.SheduleId,
		IsGroup:   shedule.IsGroup,
	}
	// TODO: вынести количество недель в переменную, либо автоматически определять конец
	for week := 1; week < 21; week++ {
		sh.Week = week
		err := sh.DownloadById(true)
		if err != nil {
			return err
		}
		_, _, err = ssau_parser.UpdateSchedule(bot.DB, sh)
		if err != nil {
			return err
		}
	}

	return nil
}

// Создать условие группы/преподавателя
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
		case "mil":
			type_emoji = "🗿"
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

	str += "------------------------------------------\n"
	return str, nil
}
