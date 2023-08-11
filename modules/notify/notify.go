package notify

import (
	"log"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
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
	NextLesson NotifyType = "nextnote"
	NextDay    NotifyType = "nextday"
	NextWeek   NotifyType = "nextweek"
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

	// Отсеиваем последние пары дня
	last := ssau_parser.Diff(completed, next)

	for _, l := range last {
		var next_lesson database.Lesson
		if _, err := db.
			Where(
				"groupid = ? and begin > ?",
				l.GroupId, l.Begin.Format("2006-01-02 15:04:03"),
			).
			Asc("begin").
			Get(&next_lesson); err != nil {
			return nil, err
		}
		// Разделяем, какие пары на этой неделе, какие на следующей

		_, nl_week := next_lesson.Begin.ISOWeek()
		_, now_week := now.ISOWeek()
		note := Notify{
			IsGroup:   true,
			SheduleId: next_lesson.GroupId,
			Lesson:    next_lesson,
		}
		if nl_week == now_week {
			note.NotifyType = NextDay
		} else {
			note.NotifyType = NextWeek
		}
		if !slices.Contains(notify, note) {
			notify = append(notify, note)
		}

	}
	return notify, nil
}

// Текст уведомления о следующей паре
func StrNext(db *xorm.Engine, note Notify) (string, error) {
	// TODO: перескакивать окна
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

// Текст уведомления о следующем дне
func StrNextDay(bot *tg.Bot, note Notify) (string, error) {
	begin := note.Lesson.Begin
	day := time.Date(begin.Year(), begin.Month(), begin.Day(), 0, 0, 0, 0, begin.Location())
	shedules := []database.ShedulesInUser{
		{
			IsGroup:   true,
			SheduleId: note.Lesson.GroupId,
		},
	}
	lessons, err := bot.GetLessons(shedules, day)
	if err != nil {
		return "", err
	}
	if len(lessons) != 0 {
		pairs := tg.GroupPairs(lessons)
		dayStr, err := bot.StrDayShedule(pairs, shedules[0].IsGroup)
		if err != nil {
			return "", err
		}
		str := "Сегодня больше ничего нет\n"
		str += "Следующие занятия в " + tg.DayStr(day) + ":\n\n" + dayStr
		return str, nil
	}
	return "", nil
}

// Рассылка всех уведомлений
func Mailing(bot *tg.Bot, notes []Notify, now time.Time) {
	var ids []int64
	for _, note := range notes {

		var users []database.TgUser
		query := database.ShedulesInUser{
			IsGroup:   note.IsGroup,
			SheduleId: note.SheduleId,
		}
		var txt string
		var err error
		switch note.NotifyType {
		case NextLesson:
			query.NextNote = true
			txt, err = StrNext(bot.DB, note)
		case NextDay:
			query.NextDay = true
			txt, err = StrNextDay(bot, note)
		case NextWeek:
			query.NextWeek = true
		}
		if err != nil {
			log.Println(err)
		}
		if err := bot.DB.
			UseBool(string(note.NotifyType)).
			Table("ShedulesInUser").
			Cols("tgid").
			Join("INNER", "tguser", "tguser.l9id = ShedulesInUser.l9id").
			Where("subgroup in (0, ?)", note.Lesson.SubGroup).
			Find(&users, &query); err != nil {
			log.Println(err)
		}
		for _, user := range users {
			if !slices.Contains(ids, user.TgId) {
				msg := tgbotapi.NewMessage(user.TgId, txt)
				msg.ParseMode = tgbotapi.ModeHTML
				m, err := bot.TG.Send(msg)
				if err != nil {
					log.Println(err)
				}
				if _, err := bot.DB.InsertOne(database.TempMsg{
					TgId:      m.Chat.ID,
					MessageId: m.MessageID,
					Destroy:   note.Lesson.Begin.Add(15 * time.Minute),
				}); err != nil {
					log.Println(err)
				}
				ids = append(ids, user.TgId)
			}
		}
	}
}

// Удаление временных сообщений
func ClearTemp(bot *tg.Bot, now time.Time) {
	var temp []database.TempMsg
	if err := bot.DB.Where("destroy < ?", now.Format("2006-01-02 15:04:03")).Find(&temp); err != nil {
		log.Println(err)
	}
	for _, msg := range temp {
		del := tgbotapi.NewDeleteMessage(msg.TgId, msg.MessageId)
		if _, err := bot.TG.Request(del); err != nil {
			log.Println(err)
		}
		if _, err := bot.DB.Delete(&msg); err != nil {
			log.Println(err)
		}
	}
}
