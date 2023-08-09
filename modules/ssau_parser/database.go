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
	// Проверяем преподавателей и группы в БД
	for _, l := range sh.Uncovered {
		_, err := CheckGroupOrTeacher(db, WeekShedule{
			IsGroup:   false,
			SheduleId: l.TeacherId,
		})
		if err != nil {
			return nil, nil, err
		}

		_, err = CheckGroupOrTeacher(db, WeekShedule{
			IsGroup:   true,
			SheduleId: l.GroupId,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	first_new := sh.Uncovered[0]
	_, week := first_new.Begin.ISOWeek()
	var old []database.Lesson
	var condition string
	if sh.IsGroup {
		condition = "groupid = ?"
	} else {
		condition = "teacherid = ?"
	}
	if err := db.Where("WEEK(`Begin`) = ? AND "+condition, week, sh.SheduleId).Asc("Begin").Find(&old); err != nil {
		return nil, nil, err
	}
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
	if sh.IsGroup {
		gr := database.Group{GroupId: sh.SheduleId}
		if _, err := db.Get(&gr); err != nil {
			return nil, nil, err
		}
		gr.LastUpd = time.Now()
		if _, err := db.ID(gr.GroupId).Update(gr); err != nil {
			return nil, nil, err
		}
	} else {
		t := database.Teacher{TeacherId: sh.SheduleId}
		if _, err := db.Get(&t); err != nil {
			return nil, nil, err
		}
		t.LastUpd = time.Now()
		if _, err := db.ID(t.TeacherId).Update(t); err != nil {
			return nil, nil, err
		}
	}
	return add, del, nil
}

func isGroupExists(db *xorm.Engine, groupId int64) (database.Group, error) {
	var groups []database.Group
	err := db.Find(&groups, database.Group{GroupId: groupId})
	if err != nil {
		return database.Group{}, err
	}
	if len(groups) == 0 {
		groups = append(groups, database.Group{})
	}
	return groups[0], nil
}

func isTeacherExists(db *xorm.Engine, teacherId int64) (database.Teacher, error) {
	var teachers []database.Teacher
	err := db.Find(&teachers, database.Teacher{TeacherId: teacherId})
	if err != nil {
		return database.Teacher{}, err
	}
	if len(teachers) == 0 {
		teachers = append(teachers, database.Teacher{})
	}
	return teachers[0], nil
}

// Проверка наличия группы или преподавателя в БД и добавление при необходимости
// Возвращает истину, если группы/преподавателя раньше не было
// TODO: Добавить проверку изменений в полях данных
func CheckGroupOrTeacher(db *xorm.Engine, sh WeekShedule) (bool, error) {
	if sh.IsGroup {
		group, err := isGroupExists(db, sh.SheduleId)
		if err != nil {
			return false, err
		}
		nilGr := database.Group{}
		if group == nilGr {
			sh.Week = 1
			sh.DownloadById(false)
			group := database.Group{
				GroupId:   sh.SheduleId,
				GroupName: sh.FullName,
				SpecName:  sh.SpecName,
			}
			if _, err := db.InsertOne(group); err != nil {
				return false, err
			}
			return true, nil
		} else if group.LastUpd.IsZero() {
			return true, nil
		}
	} else {
		teacher, err := isTeacherExists(db, sh.SheduleId)
		if err != nil {
			return false, err
		}
		nilT := database.Teacher{}
		if teacher == nilT {
			sh.Week = 1
			sh.DownloadById(false)
			teacher := ParseTeacherName(sh.FullName)
			teacher.TeacherId = sh.SheduleId
			teacher.SpecName = sh.SpecName
			if _, err := db.InsertOne(teacher); err != nil {
				return false, err
			}
			return true, nil
		} else if teacher.LastUpd.IsZero() {
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
		ShortName: strings.Join(short_name, ".") + ".",
	}
	return teacher
}
