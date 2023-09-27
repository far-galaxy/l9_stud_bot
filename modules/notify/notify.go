package notify

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssauparser"
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

type NoteType string

const (
	NextLesson NoteType = "nextnote"
	NextDay    NoteType = "nextday"
	NextWeek   NoteType = "nextweek"
	Changes    NoteType = "changes"
	Military   NoteType = "mil"
)

type Notify struct {
	NoteType
	IsGroup   bool
	SheduleID int64
	Lesson    database.Lesson
}

// Поиск следующей пары, дня, недели
func CheckNext(db *xorm.Engine, now time.Time) ([]Notify, error) {
	now = now.Truncate(time.Minute)
	var completed []database.Lesson
	if err := db.
		Desc("Begin").
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
			NoteType:  NextLesson,
			IsGroup:   true,
			SheduleID: n.GroupId,
			Lesson:    n,
		})
		if n.TeacherId != 0 {
			notify = append(notify, Notify{
				NoteType:  NextLesson,
				IsGroup:   false,
				SheduleID: n.TeacherId,
				Lesson:    n,
			})
		}

	}

	// Отсеиваем последние пары дня
	last := ssauparser.Diff(completed, next)

	for _, l := range last {
		var nextLesson database.Lesson
		if _, err := db.
			Where(
				"groupid = ? and begin > ?",
				l.GroupId, l.Begin.Format("2006-01-02 15:04:05"),
			).
			Asc("begin").
			Get(&nextLesson); err != nil {
			return nil, err
		}
		// Разделяем, какие пары на этой неделе, какие на следующей

		_, nlWeek := nextLesson.Begin.ISOWeek()
		_, nowWeek := now.ISOWeek()
		note := Notify{
			IsGroup:   true,
			SheduleID: nextLesson.GroupId,
			Lesson:    nextLesson,
		}
		if nlWeek == nowWeek {
			note.NoteType = NextDay
		} else {
			note.NoteType = NextWeek
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
			TeacherId: note.SheduleID,
		}
		if err := db.Find(&pair, query); err != nil {
			return "", err
		}
	} else {
		pair = append(pair, note.Lesson)
	}

	str := "Сейчас будет:\n\n"
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
	shedule := database.ShedulesInUser{
		IsGroup:   true,
		SheduleId: note.Lesson.GroupId,
	}
	lessons, err := bot.GetLessons(shedule, day, 32)
	if err != nil {
		return "", err
	}
	if len(lessons) != 0 {
		pairs := tg.GroupPairs(lessons)
		dayStr, err := bot.StrDayShedule(pairs, shedule.IsGroup)
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
			SheduleId: note.SheduleID,
		}
		var txt string
		var err error
		var tempTime time.Time
		switch note.NoteType {
		case NextLesson:
			query.NextNote = true
			txt, err = StrNext(bot.DB, note)
			tempTime = note.Lesson.Begin.Add(15 * time.Minute)
		case NextDay:
			query.NextDay = true
			txt, err = StrNextDay(bot, note)
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
			UseBool(string(note.NoteType)).
			Table("ShedulesInUser").
			Cols("TgId", "TgUser.L9Id").
			Join("INNER", "TgUser", "TgUser.L9Id = ShedulesInUser.L9Id").
			// Where(condition, note.Lesson.SubGroup).
			Find(&users, &query); err != nil {
			log.Println(err)
		}
		for i, user := range users {
			if slices.Contains(ids, user.TgId) {
				continue
			}
			if note.NoteType != NextWeek {
				m, err := bot.SendMsg(&users[i], txt, tg.GeneralKeyboard(true))
				if err != nil {
					bot.CheckBlocked(err, user)
				} else {
					if note.NoteType == NextDay {
						getNextDayTemp(user, bot, &tempTime, note)
					}
					AddTemp(m, tempTime, bot)
				}
			} else {
				if err := sendNextWeek(bot, note, &users[i], now); err != nil {
					log.Println(err)

					continue
				}
			}
			ids = append(ids, user.TgId)

		}
	}
}

// Рассылка уведомлений о следующей неделе
func sendNextWeek(bot *tg.Bot, note Notify, user *database.TgUser, now time.Time) error {
	if err := bot.GetWeekSummary(
		note.Lesson.Begin,
		user,
		database.ShedulesInUser{},
		-1,
		true,
		"На этой неделе больше ничего нет\n\nНа фото расписание на следующую неделю",
	); err != nil {
		return err
	}

	return bot.CreateICS(
		now,
		user,
		database.ShedulesInUser{},
		true,
		-1,
	)
}

// Получить время удаления уведомления о следующем дне
func getNextDayTemp(user database.TgUser, bot *tg.Bot, tempTime *time.Time, note Notify) {
	shInfo := database.ShedulesInUser{
		L9Id: user.L9Id,
	}
	_, err := bot.DB.Get(&shInfo)
	if err != nil {
		bot.Debug.Println(err)

		return
	}
	dt := -1 * shInfo.FirstTime
	*tempTime = note.Lesson.Begin.Add(time.Duration(dt) * time.Minute)
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
	for i, msg := range temp {
		del := tgbotapi.NewDeleteMessage(msg.TgId, msg.MessageId)
		if _, err := bot.TG.Request(del); err != nil {
			log.Println(err)
		}
		if _, err := bot.DB.Delete(&temp[i]); err != nil {
			log.Println(err)
		}
	}
}

var firstMailQuery = `SELECT t.TgId, a.LessonId, u.FirstTime
FROM ShedulesInUser u
JOIN (SELECT GroupId, MIN(Begin) as Begin FROM Lesson WHERE DATE(Begin) = DATE('%s') GROUP BY GroupId) l 
ON '%s' = DATE_SUB(l.Begin, INTERVAL u.FirstTime MINUTE) AND u.SheduleId = l.GroupId
JOIN (SELECT LessonId, Type, GroupId, Begin FROM Lesson WHERE DATE(Begin) = date('%s')) a
ON a.GroupId = l.GroupId AND a.Begin=l.Begin
JOIN TgUser t ON u.L9ID = t.L9ID
WHERE u.First = true AND (a.Type != "mil" OR (a.Type = "mil" AND u.Military = true));`

// Рассылка сообщений о начале занятий
func FirstMailing(bot *tg.Bot, now time.Time) {
	now = now.Truncate(time.Minute)
	nowStr := now.Format("2006-01-02 15:04:05")
	res, err := bot.DB.Query(fmt.Sprintf(firstMailQuery, nowStr, nowStr, nowStr))
	if err != nil {
		log.Println(err)
	}
	for _, r := range res {
		lid, _ := strconv.ParseInt(string(r["LessonId"]), 0, 64)
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
		str += fmt.Sprintf("Через %s минут начнутся занятия\n\nПервая пара:\n", r["FirstTime"])
		pair, err := tg.PairToStr([]database.Lesson{lesson}, bot.DB, true)
		if err != nil {
			log.Println(err)
		}
		str += pair
		user, _ := strconv.ParseInt(string(r["TgId"]), 0, 64)
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
