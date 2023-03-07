package ssau_parser

import (
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"xorm.io/xorm"
)

func uploadShedule(db *xorm.Engine, sh Shedule) {
	for _, line := range sh.Lessons {
		for _, lesson := range line {
			var pair database.Lesson
			for _, subLesson := range lesson.SubLessons {
				pair = database.Lesson{
					Begin:     lesson.Begin,
					End:       lesson.End,
					Type:      subLesson.Type,
					Name:      subLesson.Name,
					TeacherId: subLesson.TeacherId,
					Place:     subLesson.Place,
					Comment:   subLesson.Comment,
					SubGroup:  subLesson.SubGroup,
				}
				for _, groupId := range subLesson.GroupId {
					pair.GroupId = groupId
					db.InsertOne(pair)
				}
			}
		}
	}
}
