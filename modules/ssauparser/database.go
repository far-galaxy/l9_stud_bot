package ssauparser

import (
	"strings"

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
		if l.TeacherId != 0 {
			_, err := CheckGroupOrTeacher(db, WeekShedule{
				IsGroup:   false,
				SheduleID: l.TeacherId,
			})
			if err != nil {
				return nil, nil, err
			}
		}

		_, err := CheckGroupOrTeacher(db, WeekShedule{
			IsGroup:   true,
			SheduleID: l.GroupId,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	firstNew := sh.Uncovered[0]
	_, week := firstNew.Begin.ISOWeek()
	var oldLessons []database.Lesson
	var newLessons []database.Lesson
	// Удаляем всё, что не относится к данной группе
	for _, l := range sh.Uncovered {
		if (sh.IsGroup && l.GroupId == sh.SheduleID) ||
			(!sh.IsGroup && l.TeacherId == sh.SheduleID) {
			newLessons = append(newLessons, l)
		}
	}
	var condition string
	if sh.IsGroup {
		condition = "groupid = ?"
	} else {
		condition = "teacherid = ?"
	}
	if err := db.Where("WEEK(`Begin`) = ? AND "+condition, week, sh.SheduleID).Asc("Begin").Find(&oldLessons); err != nil {
		return nil, nil, err
	}
	add, del := Compare(newLessons, oldLessons)
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

func isGroupExists(db *xorm.Engine, groupID int64) (database.Group, error) {
	var groups []database.Group
	err := db.Find(&groups, database.Group{GroupId: groupID})
	if err != nil {
		return database.Group{}, err
	}
	if len(groups) == 0 {
		groups = append(groups, database.Group{})
	}

	return groups[0], nil
}

// Проверка наличия группы или преподавателя в БД и добавление при необходимости
// Возвращает истину, если группы/преподавателя раньше не было
// TODO: Добавить проверку изменений в полях данных
func CheckGroupOrTeacher(db *xorm.Engine, sh WeekShedule) (bool, error) {
	if sh.IsGroup {
		group, err := isGroupExists(db, sh.SheduleID)
		if err != nil {
			return false, err
		}
		nilGr := database.Group{}
		if group == nilGr {
			sh.Week = 1
			if err := sh.DownloadByID(false); err != nil {
				return false, err
			}
			group := database.Group{
				GroupId:   sh.SheduleID,
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
		var teachers []database.Teacher
		tc, err := db.FindAndCount(&teachers, database.Teacher{TeacherId: sh.SheduleID})
		if err != nil {
			return false, err
		}
		if tc == 0 {
			sh.Week = 1
			if err := sh.DownloadByID(false); err != nil {
				return false, err
			}
			teacher := ParseTeacherName(sh.FullName)
			teacher.TeacherId = sh.SheduleID
			teacher.SpecName = sh.SpecName
			if _, err := db.InsertOne(teacher); err != nil {
				return false, err
			}

			return true, nil
		} else if teachers[0].LastUpd.IsZero() {
			return true, nil
		}

	}

	return false, nil
}

func ParseTeacherName(fullName string) database.Teacher {
	name := strings.Split(fullName, " ")
	var shortName []string
	for _, a := range name[1:] {
		shortName = append(shortName, a[:2])
	}
	teacher := database.Teacher{
		FirstName: name[0],
		LastName:  strings.Join(name[1:], " "),
		ShortName: strings.Join(shortName, ".") + ".",
	}

	return teacher
}
