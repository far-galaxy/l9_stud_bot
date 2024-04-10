package notify

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/slices"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
	"stud.l9labs.ru/bot/modules/tg"
	"xorm.io/xorm"
)

type Next struct {
	Lesson []database.Lesson
	Day    []database.Lesson
	Week   []database.Lesson
}

// Поиск следующей пары, дня, недели
func CheckNext(db *xorm.Engine, now time.Time) ([]api.Notify, error) {
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
	var notify []api.Notify
	for _, n := range next {
		notify = append(notify, api.Notify{
			NoteType:  api.NextLesson,
			IsGroup:   true,
			SheduleID: n.GroupId,
			Lesson:    n,
		})
		if n.TeacherId != 0 {
			notify = append(notify, api.Notify{
				NoteType:  api.NextLesson,
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
		note := api.Notify{
			IsGroup:   true,
			SheduleID: nextLesson.GroupId,
			Lesson:    nextLesson,
		}
		if nlWeek == nowWeek {
			note.NoteType = api.NextDay
		} else {
			note.NoteType = api.NextWeek
		}
		if !slices.Contains(notify, note) {
			notify = append(notify, note)
		}

	}

	return notify, nil
}

// Текст уведомления о следующей паре
func StrNext(db *xorm.Engine, note api.Notify) (string, error) {
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
func StrNextDay(bot *tg.Bot, note api.Notify) (string, error) {
	begin := note.Lesson.Begin
	day := time.Date(begin.Year(), begin.Month(), begin.Day(), 0, 0, 0, 0, begin.Location())
	shedule := database.Schedule{
		IsGroup:    true,
		ScheduleID: note.Lesson.GroupId,
	}
	lessons, err := api.GetDayLessons(bot.DB, shedule, day)
	if err != nil {
		return "", err
	}
	if len(lessons) != 0 {
		pairs := api.GroupPairs(lessons)
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
func Mailing(bot *tg.Bot, notes []api.Notify) {
	var ids []int64
	for _, note := range notes {
		if note.SheduleID == 0 {
			continue
		}

		var txt string
		var err error
		var tempTime time.Time
		switch note.NoteType {
		case api.NextLesson:
			txt, err = StrNext(bot.DB, note)
			tempTime = note.Lesson.Begin.Add(120 * time.Minute)
		case api.NextDay:
			txt, err = StrNextDay(bot, note)
		}
		if err != nil {
			log.Println(err)
		}
		// TODO: проработать разные подгруппы
		users, err := api.GetUserForNote(bot.DB, note)
		if err != nil {
			log.Println(err)
		}
		for i, user := range users {
			if slices.Contains(ids, user.TgId) {
				continue
			}
			if note.NoteType != api.NextWeek {
				var markup tgbotapi.InlineKeyboardMarkup
				if note.NoteType == api.NextLesson {
					markup = tgbotapi.InlineKeyboardMarkup{
						InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
							{
								tgbotapi.NewInlineKeyboardButtonData(
									"Добавить заметку",
									fmt.Sprintf("note_%d", note.Lesson.LessonId),
								),
							},
						}}

				}
				m, err := bot.SendMsg(&users[i], txt, markup)
				if err != nil {
					bot.CheckBlocked(err, user)
				} else {
					if note.NoteType == api.NextDay {
						getNextDayTemp(user, bot, &tempTime, note)
					}
					AddTemp(m, tempTime, bot)
				}
			} else {
				if err := sendNextWeek(bot, note, &users[i]); err != nil {
					log.Println(err)

					continue
				}
			}
			ids = append(ids, user.TgId)

		}
	}
}

// Рассылка уведомлений о следующей неделе
func sendNextWeek(bot *tg.Bot, note api.Notify, user *database.TgUser) error {
	if note.Lesson.Begin.IsZero() {
		return fmt.Errorf("null lesson")
	}
	sh := database.Schedule{
		TgUser:     user,
		IsPersonal: true,
	}
	_, err := bot.GetWeekSummary(
		note.Lesson.Begin,
		sh,
		-1,
		"На этой неделе больше ничего нет\n\nНа фото расписание на следующую неделю",
	)

	return err
}

// Получить время удаления уведомления о следующем дне
func getNextDayTemp(user database.TgUser, bot *tg.Bot, tempTime *time.Time, note api.Notify) {
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
	temp, err := api.GetExpiredNotifies(bot.DB, now)
	HandleErr(err)
	for i, msg := range temp {
		del := tgbotapi.NewDeleteMessage(msg.TgId, msg.MessageId)
		_, err := bot.TG.Request(del)
		HandleErr(err)

		_, err = bot.DB.Delete(&temp[i])
		HandleErr(err)
	}
}

// Рассылка сообщений о начале занятий
func FirstMailing(bot *tg.Bot, now time.Time) {
	res, err := api.GetFirstLessonNote(bot.DB, now)
	if err != nil {
		log.Println(err)

		return
	}
	for _, r := range res {
		lesson, err := api.GetLesson(bot.DB, r.LessonID)
		if err != nil {
			log.Println(err)

			return
		}
		var str string
		if now.Hour() >= 16 {
			str = "Добрый вечер 🌆\n"
		} else if now.Hour() >= 11 {
			str = "Добрый день 🌞\n"
		} else {
			str = "Доброе утро 🌅\n"
		}
		str += fmt.Sprintf("Через %s минут начнутся занятия\n\nПервая пара:\n", r.Time)
		pair, err := tg.PairToStr([]database.Lesson{lesson}, bot.DB, true)
		if err != nil {
			log.Println(err)
		}
		str += pair
		mail := tgbotapi.NewMessage(r.TgID, str)
		msg, err := bot.TG.Send(mail)
		if err != nil {
			log.Println(err)

			continue
		}
		AddTemp(msg, lesson.Begin.Add(15*time.Minute), bot)
	}
}

// Логирование некритических ошибок
func HandleErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
