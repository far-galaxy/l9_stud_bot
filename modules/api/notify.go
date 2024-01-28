package api

import (
	"fmt"
	"strconv"
	"time"

	"stud.l9labs.ru/bot/modules/database"
	"xorm.io/xorm"
)

type NoteType string

const (
	NextLesson NoteType = "nextnote"
	NextDay    NoteType = "nextday"
	NextWeek   NoteType = "nextweek"
)

type Notify struct {
	NoteType
	IsGroup   bool
	SheduleID int64
	Lesson    database.Lesson
}

func GetUserForNote(db *xorm.Engine, note Notify) ([]database.TgUser, error) {
	var users []database.TgUser
	query := database.ShedulesInUser{
		IsGroup:   note.IsGroup,
		SheduleID: note.SheduleID,
	}
	switch note.NoteType {
	case NextLesson:
		query.NextNote = true
	case NextDay:
		query.NextDay = true
	case NextWeek:
		query.NextWeek = true
	default:
		return users, nil
	}

	err := db.UseBool(string(note.NoteType), "IsGroup").
		Table("ShedulesInUser").
		Cols("TgID", "TgUser.L9ID").
		Join("INNER", "TgUser", "TgUser.L9ID = ShedulesInUser.L9ID").
		Find(&users, &query)

	return users, err
}

func GetExpiredNotifies(db *xorm.Engine, now time.Time) ([]database.TempMsg, error) {
	var temp []database.TempMsg
	err := db.Where("destroy <= ?", now.Format("2006-01-02 15:04:05")).Find(&temp)

	return temp, err
}

// Почтовая рассыка о начале занятий
type FirstMail struct {
	TgID     int64  // Получатель
	LessonID int64  // Первое занятие
	Time     string // Время до начала занятий (чисто для сообщения, поэтому формат string сойдёт)
}

var firstMailQuery = `SELECT t.TgID, a.LessonID, u.FirstTime
FROM ShedulesInUser u
JOIN (SELECT GroupID, MIN(Begin) as Begin FROM Lesson WHERE DATE(Begin) = DATE('%s') GROUP BY GroupID) l 
ON '%s' = DATE_SUB(l.Begin, INTERVAL u.FirstTime MINUTE) AND u.SheduleID = l.GroupID
JOIN (SELECT LessonID, Type, GroupID, Begin FROM Lesson WHERE DATE(Begin) = date('%s')) a
ON a.GroupID = l.GroupID AND a.Begin=l.Begin
JOIN TgUser t ON u.L9ID = t.L9ID
WHERE u.First = true AND (a.Type != "mil" OR (a.Type = "mil" AND u.Military = true));`

func GetFirstLessonNote(db *xorm.Engine, now time.Time) ([]FirstMail, error) {
	var mailing []FirstMail
	nowStr := now.Format("2006-01-02 15:04:05")
	res, err := db.Query(fmt.Sprintf(firstMailQuery, nowStr, nowStr, nowStr))
	if err != nil {
		return mailing, err
	}
	for _, r := range res {
		var mail FirstMail
		tgID, _ := strconv.ParseInt(string(r["TgID"]), 0, 64)
		mail.TgID = tgID
		mail.Time = string(r["FirstTime"])
		lid, _ := strconv.ParseInt(string(r["LessonID"]), 0, 64)
		mail.LessonID = lid

		mailing = append(mailing, mail)
	}

	return mailing, nil
}
