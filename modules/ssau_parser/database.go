package ssau_parser

import (
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"xorm.io/xorm"
)

// Согласование недельного расписания с БД
// Возвращает соответственно добавленные и удалённые занятия
func UpdateSchedule(db *xorm.Engine, sh WeekShedule) ([]database.Lesson, []database.Lesson, error) {
	if _, err := CheckGroupOrTeacher(db, sh); err != nil {
		return nil, nil, err
	}
	if len(sh.Uncovered) == 0 {
		return nil, nil, nil
	}
	first_new := sh.Uncovered[0]
	_, week := first_new.Begin.ISOWeek()
	var old []database.Lesson
	db.Where("WEEK(`Begin`) = ?", week).Asc("Begin").Find(&old)
	add, del := Compare(sh.Uncovered, old)
	if len(add) > 0 {
		if _, err := db.Insert(add); err != nil {
			return nil, nil, err
		}
	}
	if len(del) > 0 {
		var ids []int64
		for _, d := range del {
			ids = append(ids, d.LessonId)
		}
		if _, err := db.In("lessonid", ids).Delete(&database.Lesson{}); err != nil {
			return nil, nil, err
		}
	}
	// Обновляем время обновления
	if len(add) > 0 || len(del) > 0 {
		if sh.IsGroup {
			gr := database.Group{GroupId: sh.SheduleId}
			if _, err := db.Get(&gr); err != nil {
				return nil, nil, err
			}
			gr.LastUpd = time.Now()
			if _, err := db.Update(gr); err != nil {
				return nil, nil, err
			}
		} else {
			var t database.Teacher
			if err := db.Find(&t, &database.Teacher{TeacherId: sh.SheduleId}); err != nil {
				return nil, nil, err
			}
			t.LastUpd = time.Now()
			if _, err := db.Update(t); err != nil {
				return nil, nil, err
			}
		}
	}
	return add, del, nil
}

func isGroupExists(db *xorm.Engine, groupId int64) (bool, error) {
	var exists []database.Group
	err := db.Find(&exists, database.Group{GroupId: groupId})
	if err != nil {
		return false, err
	}

	return len(exists) == 1, nil
}

func isTeacherExists(db *xorm.Engine, teacherId int64) (bool, error) {
	var exists []database.Teacher
	err := db.Find(&exists, database.Teacher{TeacherId: teacherId})
	if err != nil {
		return false, err
	}

	return len(exists) == 1, nil
}

// Проверка наличия группы или преподавателя в БД и добавление при необходимости
// Возвращает истину, если группы/преподавателя раньше не было
// TODO: Добавить проверку изменений в полях данных
func CheckGroupOrTeacher(db *xorm.Engine, sh WeekShedule) (bool, error) {
	if sh.IsGroup {
		exists, err := isGroupExists(db, sh.SheduleId)
		if err != nil {
			return false, err
		}

		if !exists {
			group := database.Group{
				GroupId:   sh.SheduleId,
				GroupName: sh.FullName,
				SpecName:  sh.SpecName,
			}
			db.InsertOne(group)
			return true, nil
		}
	} else {
		exists, err := isTeacherExists(db, sh.SheduleId)
		if err != nil {
			return false, err
		}

		if !exists {
			teacher := ParseTeacherName(sh.FullName)
			teacher.TeacherId = sh.SheduleId
			teacher.SpecName = sh.SpecName
			db.InsertOne(teacher)
			return true, nil
		}

	}
	return false, nil
}

func ParseTeacherName(fullName string) database.Teacher {
	name := strings.Split(fullName, " ")
	var short_name []string
	for _, a := range name[1:] {
		short_name = append(short_name, a[:2])
	}
	teacher := database.Teacher{
		FirstName: name[0],
		LastName:  strings.Join(name[1:], " "),
		ShortName: strings.Join(short_name, ". ") + ".",
	}
	return teacher
}
