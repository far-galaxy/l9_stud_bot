package ssau_parser

import (
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"xorm.io/xorm"
)

// Согласование недельного расписания с БД
// Возвращает соответственно добавленные и удалённые занятия
func UpdateSchedule(db *xorm.Engine, sh WeekShedule) ([]database.Lesson, []database.Lesson, error) {
	if err := checkGroupOrTeacher(db, sh); err != nil {
		return nil, nil, err
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
// TODO: Добавить проверку изменений в полях данных
func checkGroupOrTeacher(db *xorm.Engine, sh WeekShedule) error {
	if sh.IsGroup {
		exists, err := isGroupExists(db, sh.SheduleId)
		if err != nil {
			return err
		}

		if !exists {
			group := database.Group{
				GroupId:   sh.SheduleId,
				GroupName: sh.FullName,
				SpecName:  sh.SpecName,
			}
			db.InsertOne(group)
		}
	} else {
		exists, err := isTeacherExists(db, sh.SheduleId)
		if err != nil {
			return err
		}

		if !exists {
			teacher := ParseTeacherName(sh.FullName)
			teacher.TeacherId = sh.SheduleId
			teacher.SpecName = sh.SpecName
			db.InsertOne(teacher)
		}

	}
	return nil
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
