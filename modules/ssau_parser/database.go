package ssau_parser

import (
	"log"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"xorm.io/xorm"
)

func UploadShedule(db *xorm.Engine, sh WeekShedule) error {
	err := addGroupOrTeacher(db, sh)
	if err != nil {
		return err
	}

	var pairs []database.Lesson
	for _, line := range sh.Lessons {
		for _, lesson := range line {
			var pair database.Lesson
			for _, subLesson := range lesson.Lessons {
				pair = database.Lesson{
					Begin:     lesson.Begin,
					End:       lesson.End,
					Type:      subLesson.Type,
					Name:      subLesson.Name,
					TeacherId: subLesson.TeacherId,
				}

				exists, err := isTeacherExists(db, subLesson.TeacherId)
				if err != nil {
					return err
				}

				if !exists && subLesson.TeacherId != 0 {
					uri := GenerateUri(subLesson.TeacherId, true)
					doc, err := DownloadShedule(uri, sh.Week)
					if err != nil {
						return err
					}
					var gr WeekShedule
					gr.IsGroup = false
					gr.SheduleId = subLesson.TeacherId
					GetSheduleInfo(doc.Doc, &gr)
					addGroupOrTeacher(db, gr)
				}

				for _, groupId := range subLesson.GroupId {
					pair.GroupId = groupId

					exists, err := isGroupExists(db, groupId)
					if err != nil {
						return err
					}

					if !exists {
						uri := GenerateUri(groupId, false)
						doc, err := DownloadShedule(uri, sh.Week)
						if err != nil {
							return err
						}
						var gr WeekShedule
						gr.IsGroup = true
						gr.SheduleId = groupId
						GetSheduleInfo(doc.Doc, &gr)
						addGroupOrTeacher(db, gr)
					}

					var existsLessons []database.Lesson
					err = db.Find(&existsLessons, pair)
					if err != nil {
						return err
					}
					if len(existsLessons) == 0 {
						pair.NumInShedule = lesson.NumInShedule
						pair.Place = subLesson.Place
						pair.Comment = subLesson.Comment
						pair.SubGroup = subLesson.SubGroup
						pairs = append(pairs, pair)
					}
				}
			}
		}
	}
	if len(pairs) > 0 {
		_, err := db.Insert(&pairs)
		return err
	}
	return nil
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

func addGroupOrTeacher(db *xorm.Engine, sh WeekShedule) error {
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
			name := strings.Split(sh.FullName, " ")
			log.Println(name)
			teacher := database.Teacher{
				TeacherId: sh.SheduleId,
				LastName:  name[0],
				FirstName: strings.Join(name[1:], " "),
				SpecName:  sh.SpecName,
			}
			db.InsertOne(teacher)
		}

	}
	return nil
}
