package ssau_parser

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"crypto/md5"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
)

// Раскрытие недельного расписания в список занятий для базы данных и сравнения
func UncoverShedule(sh WeekShedule) []database.Lesson {
	var lessons []database.Lesson
	for y, line := range sh.Lessons {
		for x, pair := range line {
			for _, lesson := range pair.Lessons {
				for i, gId := range lesson.GroupId {
					if len(lesson.TeacherId) == 0 {
						lesson.TeacherId = append(lesson.TeacherId, 0)
					}
					for _, tId := range lesson.TeacherId {
						l := database.Lesson{
							NumInShedule: pair.NumInShedule,
							Type:         lesson.Type,
							Name:         lesson.Name,
							Begin:        pair.Begin,
							End:          pair.End,
							TeacherId:    tId,
							GroupId:      gId,
							Place:        lesson.Place,
							Comment:      lesson.Comment,
							SubGroup:     int64(lesson.SubGroup[i]),
						}
						l.Hash = Hash(l)
						l.LessonId = int64(y*len(line) + x)
						lessons = append(lessons, l)
					}
				}
			}
		}
	}
	return lessons
}

// Сравнивание списков занятий на предмет добавления и удаления
func Compare(new []database.Lesson, old []database.Lesson) ([]database.Lesson, []database.Lesson) {
	added := Diff(new, old)
	deleted := Diff(old, new)
	return added, deleted
}

// Проверка занятий, которые появились в eden и отсутствуют в dzwa
func Diff(jeden []database.Lesson, dzwa []database.Lesson) []database.Lesson {
	var diff []database.Lesson
	for _, n := range jeden {
		exists := false
		for _, o := range dzwa {
			if n.Hash == o.Hash {
				exists = true
			}
		}
		if !exists {
			diff = append(diff, n)
		}
	}
	return diff
}

func Hash(s database.Lesson) string {
	var b bytes.Buffer
	gob.NewEncoder(&b).Encode(s)
	hash := fmt.Sprintf("%x", md5.Sum(b.Bytes()))
	return hash
}
