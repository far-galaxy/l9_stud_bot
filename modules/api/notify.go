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
	Changes    NoteType = "changes"
	Military   NoteType = "mil"
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
		SheduleId: note.SheduleID,
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
		Cols("TgId", "TgUser.L9Id").
		Join("INNER", "TgUser", "TgUser.L9Id = ShedulesInUser.L9Id").
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

var firstMailQuery = `SELECT t.TgId, a.LessonId, u.FirstTime
FROM ShedulesInUser u
JOIN (SELECT GroupId, MIN(Begin) as Begin FROM Lesson WHERE DATE(Begin) = DATE('%s') GROUP BY GroupId) l 
ON '%s' = DATE_SUB(l.Begin, INTERVAL u.FirstTime MINUTE) AND u.SheduleId = l.GroupId
JOIN (SELECT LessonId, Type, GroupId, Begin FROM Lesson WHERE DATE(Begin) = date('%s')) a
ON a.GroupId = l.GroupId AND a.Begin=l.Begin
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
		tgID, _ := strconv.ParseInt(string(r["TgId"]), 0, 64)
		mail.TgID = tgID
		mail.Time = string(r["FirstTime"])
		lid, _ := strconv.ParseInt(string(r["LessonId"]), 0, 64)
		mail.LessonID = lid

		mailing = append(mailing, mail)
	}
	return mailing, nil
}
