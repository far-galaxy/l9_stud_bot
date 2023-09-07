package notify

import (
	"fmt"
	"log"
	"strconv"
	"strings"
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
		Asc("Begin").
		Find(&completed, &database.Lesson{End: now}); err != nil {
		return nil, err
	}
	if len(completed) == 0 {
		return nil, nil
	}
	num := completed[0].NumInShedule + 1

	var next []database.Lesson
	if err := db.
		Where("date(`Begin`) = ? and NumInShedule = ?", now.Format("2006-01-02"), num).
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
				l.GroupId, l.Begin.Format("2006-01-02 15:04:05"),
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
		var tempTime time.Time
		switch note.NotifyType {
		case NextLesson:
			query.NextNote = true
			txt, err = StrNext(bot.DB, note)
			tempTime = note.Lesson.Begin.Add(15 * time.Minute)
		case NextDay:
			query.NextDay = true
			txt, err = StrNextDay(bot, note)
			// TODO: установить время удаления на момент сообщения о начале пар
			tempTime = note.Lesson.Begin.Add(-60 * time.Minute)
		case NextWeek:
			query.NextWeek = true
		}
		if err != nil {
			log.Println(err)
		}
		// TODO: проработать разные подгруппы
		/*var condition string
		if note.Lesson.SubGroup == 0 {
			condition = "subgroup in (?, 1, 2)"
		} else {
			condition = "subgroup in (0, ?)"
		}*/
		if err := bot.DB.
			UseBool(string(note.NotifyType)).
			Table("ShedulesInUser").
			Cols("tgid").
			Join("INNER", "TgUser", "TgUser.l9id = ShedulesInUser.l9id").
			// Where(condition, note.Lesson.SubGroup).
			Find(&users, &query); err != nil {
			log.Println(err)
		}
		for _, user := range users {
			if !slices.Contains(ids, user.TgId) {
				if note.NotifyType != NextWeek {
					m, err := bot.SendMsg(&user, txt, tg.GeneralKeyboard(true))
					if err != nil {
						// Удаление пользователя, заблокировавшего бота
						if !strings.Contains(err.Error(), "blocked by user") {
							bot.DeleteUser(user)
							continue
						} else {
							log.Println(err)
						}
					} else {
						AddTemp(m, tempTime, bot)
					}
				} else {
					if err = bot.GetWeekSummary(
						note.Lesson.Begin,
						&user,
						database.ShedulesInUser{},
						0,
						true,
						"На этой неделе больше ничего нет\n\nНа фото расписание на следующую неделю",
					); err != nil {
						log.Println(err)
						continue
					}
				}
				ids = append(ids, user.TgId)
			}
		}
	}
}

// Добавить сообщение в список временных
func AddTemp(m tgbotapi.Message, tempTime time.Time, bot *tg.Bot) {
	temp := database.TempMsg{
		TgId:      m.Chat.ID,
		MessageId: m.MessageID,
		Destroy:   tempTime,
	}
	if _, err := bot.DB.InsertOne(temp); err != nil {
		log.Println(err)
	}
}

// Удаление временных сообщений
func ClearTemp(bot *tg.Bot, now time.Time) {
	var temp []database.TempMsg
	if err := bot.DB.Where("destroy <= ?", now.Format("2006-01-02 15:04:05")).Find(&temp); err != nil {
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

var firstMailQuery = `SELECT t.tgId, l.lessonId, u.firsttime
FROM ShedulesInUser u
JOIN (SELECT lessonid, groupid, type, min(begin) as begin FROM Lesson WHERE date(begin) = date('%s') GROUP BY groupid) l 
ON '%s' = DATE_SUB(l.Begin, INTERVAL u.firsttime MINUTE) AND u.sheduleid = l.groupid
JOIN TgUser t ON u.L9ID = t.L9ID
WHERE u.first = true AND (l.type != "mil" OR (l.type = "mil" AND u.military = true));`

// Рассылка сообщений о начале занятий
func FirstMailing(bot *tg.Bot, now time.Time) {
	now = now.Truncate(time.Minute)
	nowStr := now.Format("2006-01-02 15:04:05")
	res, err := bot.DB.Query(fmt.Sprintf(firstMailQuery, nowStr, nowStr))
	if err != nil {
		log.Println(err)
	}
	for _, r := range res {
		lid, _ := strconv.ParseInt(string(r["lessonId"]), 0, 64)
		lesson := database.Lesson{LessonId: lid}
		if _, err := bot.DB.Get(&lesson); err != nil {
			log.Println(err)
		}
		var str string
		if now.Hour() >= 16 {
			str = "Добрый вечер 🌆\n"
		} else if now.Hour() >= 11 {
			str = "Добрый день 🌞\n"
		} else {
			str = "Доброе утро 🌅\n"
		}
		str += fmt.Sprintf("Через %s минут начнутся занятия\n\nПервая пара:\n", r["firsttime"])
		pair, err := tg.PairToStr([]database.Lesson{lesson}, bot.DB, true)
		if err != nil {
			log.Println(err)
		}
		str += pair
		user, _ := strconv.ParseInt(string(r["tgId"]), 0, 64)
		mail := tgbotapi.NewMessage(user, str)
		mail.ReplyMarkup = tg.GeneralKeyboard(true)
		msg, err := bot.TG.Send(mail)
		if err != nil {
			log.Println(err)
			continue
		}
		AddTemp(msg, lesson.Begin.Add(15*time.Minute), bot)
	}
}
