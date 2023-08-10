package notify

import (
	"log"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/slices"
	"xorm.io/xorm"
)

type Next struct {
	Lesson []database.Lesson
	Day    []database.Lesson
	Week   []database.Lesson
}

type NotifyType string

const (
	NextLesson NotifyType = "next"
	NextDay    NotifyType = "day"
	NextWeek   NotifyType = "week"
	Changes    NotifyType = "changes"
	Military   NotifyType = "mil"
)

type Notify struct {
	NotifyType
	IsGroup   bool
	SheduleId int64
	Lesson    database.Lesson
}

// Поиск следующей пары, дня, недели
func CheckNext(db *xorm.Engine, now time.Time) ([]Notify, error) {
	now = now.Truncate(time.Minute)
	var completed []database.Lesson
	if err := db.
		Find(&completed, &database.Lesson{End: now}); err != nil {
		return nil, err
	}
	var nums []int
	for i := range completed {
		new_num := completed[i].NumInShedule + 1
		if !slices.Contains(nums, new_num) {
			nums = append(nums, new_num)
		}
	}
	var next []database.Lesson
	if err := db.
		Where("date(`begin`) = ?", now.Format("2006-01-02")).
		In("numinshedule", nums).
		Find(&next); err != nil {
		return nil, err
	}
	var notify []Notify
	for _, n := range next {
		notify = append(notify, Notify{
			NotifyType: NextLesson,
			IsGroup:    true,
			SheduleId:  n.GroupId,
			Lesson:     n,
		})
		if n.TeacherId != 0 {
			notify = append(notify, Notify{
				NotifyType: NextLesson,
				IsGroup:    false,
				SheduleId:  n.TeacherId,
				Lesson:     n,
			})
		}

	}
	/*
		// Отсеиваем последние пары дня
		last := ssau_parser.Diff(completed, next)

		var next_day []database.Lesson
		var next_week []database.Lesson
		for _, l := range last {
			var next_lesson []database.Lesson
			if err := db.
				Where(
					"groupid = ? and begin > ?",
					l.GroupId, now.Format("2006-01-02 15:04:03"),
				).
				Limit(1).
				Find(&next_lesson); err != nil {
				return nilNext, err
			}
			// Разделяем, какие пары на этой неделе, какие на следующей
			for _, nl := range next_lesson {
				_, nl_week := nl.Begin.ISOWeek()
				_, now_week := now.ISOWeek()
				if nl_week == now_week {
					next_day = append(next_day, nl)
				} else {
					next_week = append(next_week, nl)
				}
			}

		}*/
	return notify, nil
}

func StrNext(db *xorm.Engine, note Notify) (string, error) {
	// Подкачиваем группы и подгруппы
	var pair []database.Lesson
	if !note.IsGroup {
		query := database.Lesson{
			Begin:     note.Lesson.Begin,
			TeacherId: note.SheduleId,
		}
		if err := db.Find(&pair, query); err != nil {
			return "", err
		}
	} else {
		pair = append(pair, note.Lesson)
	}

	str := "Следующая пара:\n\n"
	strPair, err := tg.PairToStr(pair, db, note.IsGroup)
	if err != nil {
		return "", err
	}
	str += strPair
	return str, nil
}

func Mailing(bot *tg.Bot, notes []Notify) {
	var ids []int64
	for _, note := range notes {
		if note.NotifyType == NextLesson {
			var users []database.TgUser
			query := database.ShedulesInUser{
				IsGroup:   note.IsGroup,
				SheduleId: note.SheduleId,
				NextNote:  true,
			}
			if err := bot.DB.
				UseBool().
				Table("ShedulesInUser").
				Cols("tgid").
				Join("INNER", "tguser", "tguser.l9id = ShedulesInUser.l9id").
				Where("subgroup in (0, ?)", note.Lesson.SubGroup).
				Find(&users, &query); err != nil {
				log.Println(err)
			}
			for _, user := range users {
				if !slices.Contains(ids, user.TgId) {
					txt, _ := StrNext(bot.DB, note)
					msg := tgbotapi.NewMessage(user.TgId, txt)
					bot.TG.Send(msg)
					ids = append(ids, user.TgId)
				}
			}
		}
	}
}
