package api

import (
	"strconv"
	"strings"
	"time"

	"stud.l9labs.ru/bot/modules/database"
	"xorm.io/xorm"
)

// Создать условие поиска группы/преподавателя
func CreateCondition(schedule database.Schedule) string {
	var groups []string
	var teachers []string

	if schedule.IsGroup {
		groups = append(groups, strconv.FormatInt(schedule.ScheduleID, 10))
	} else {
		teachers = append(teachers, strconv.FormatInt(schedule.ScheduleID, 10))
	}

	var condition, teachersStr, groupsStr string
	if len(groups) > 0 {
		groupsStr = strings.Join(groups, ",")
		condition = "groupID in (" + groupsStr + ") "
	}
	if len(teachers) > 0 {
		if len(condition) > 0 {
			condition += " or "
		}
		teachersStr += strings.Join(teachers, ",")
		condition += "teacherID in (" + teachersStr + ") "
	}

	return condition
}

// Получить данные о занятии по его ID
func GetLesson(db *xorm.Engine, lessonID int64) (database.Lesson, error) {
	lesson := database.Lesson{LessonID: lessonID}
	_, err := db.Get(&lesson)

	return lesson, err
}

// Получить список занятий для расписания на день
func GetDayLessons(db *xorm.Engine, schedule database.Schedule, now time.Time) ([]database.Lesson, error) {

	condition := CreateCondition(schedule)

	var lessons []database.Lesson
	err := db.
		Where("DATE(`Begin`) = ?", now.Format("2006-01-02")).
		And(condition).
		OrderBy("begin").
		Find(&lessons)

	return lessons, err
}

// Получить ближайшее занятие после данного
func GetNearLesson(db *xorm.Engine, schedule database.Schedule, now time.Time) (database.Lesson, error) {

	condition := CreateCondition(schedule)

	var lesson database.Lesson
	_, err := db.
		Where("end > ?", now.Format("2006-01-02 15:04:05")).
		And(condition).
		OrderBy("begin").
		Get(&lesson)

	return lesson, err
}

// Получить занятия следующего дня
func GetNextDayLessons(db *xorm.Engine, schedule database.Schedule, now time.Time) ([]database.Lesson, error) {

	var lesson database.Lesson
	if _, err := GetNearLesson(db, schedule, now); err != nil {
		return []database.Lesson{}, err
	}

	return GetDayLessons(db, schedule, lesson.Begin)
}

// Получить занятия на неделю (указывается недели в году)
func GetWeekLessons(db *xorm.Engine, schedule database.Schedule, week int) ([]database.Lesson, error) {
	condition := CreateCondition(schedule)

	var lessons []database.Lesson
	err := db.
		Where("WEEK(`begin`, 1) = ?", week).
		And(condition).
		OrderBy("begin").
		Find(&lessons)

	return lessons, err
}

// Получить занятия на весь семестр
func GetSemesterLessons(db *xorm.Engine, schedule database.Schedule) ([]database.Lesson, error) {
	condition := CreateCondition(schedule)

	var lessons []database.Lesson
	err := db.
		Where(condition).
		OrderBy("begin").
		Find(&lessons)

	return lessons, err
}

// Получить данные о преподавателе по ID
func GetStaff(db *xorm.Engine, staffID int64) (database.Staff, error) {
	var staff database.Staff
	staff.StaffID = staffID
	_, err := db.Get(&staff)

	return staff, err
}

// Получить данные о группе по ID
func GetGroup(db *xorm.Engine, groupID int64) (database.Group, error) {
	var group database.Group
	group.GroupID = groupID
	_, err := db.Get(&group)

	return group, err
}

// Обновить информацию о группе
func UpdateGroup(db *xorm.Engine, group database.Group) error {
	_, err := db.ID(group.GroupID).Update(group)

	return err
}

// Обновить информацию о преподавателе
func UpdateStaff(db *xorm.Engine, staff database.Staff) error {
	_, err := db.ID(staff.StaffID).Update(staff)

	return err
}

// Группировка занятий по парам
func GroupPairs(lessons []database.Lesson) [][]database.Lesson {
	var shedule [][]database.Lesson
	var pair []database.Lesson

	lIDx := 0

	for lIDx < len(lessons) {
		day := lessons[lIDx].Begin
		for lIDx < len(lessons) && lessons[lIDx].Begin == day {
			pair = append(pair, lessons[lIDx])
			lIDx++
		}
		shedule = append(shedule, pair)
		pair = []database.Lesson{}
	}

	return shedule
}

func GetLastUpdate(db *xorm.Engine, sh database.Schedule) (time.Time, error) {
	var lastUpd time.Time
	if sh.IsGroup {
		group, err := GetGroup(db, sh.ScheduleID)
		if err != nil {
			return lastUpd, err
		}
		lastUpd = group.LastUpd
	} else {
		staff, err := GetStaff(db, sh.ScheduleID)
		if err != nil {
			return lastUpd, err
		}
		lastUpd = staff.LastUpd
	}

	return lastUpd, nil
}
