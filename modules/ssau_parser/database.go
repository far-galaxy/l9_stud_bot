package ssau_parser

import (
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"xorm.io/xorm"
)

func uploadShedule(db *xorm.Engine, sh Shedule) error {
	var pairs []database.Lesson
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
				}
				for _, groupId := range subLesson.GroupId {
					pair.GroupId = groupId
					var existsLessons []database.Lesson
					err := db.Find(&existsLessons, pair)
					if err != nil {
						return err
					}
					if len(existsLessons) == 0 {
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
		_, err := db.Insert(pairs)
		return err
	}
	return nil
}
