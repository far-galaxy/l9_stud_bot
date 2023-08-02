package ssau_parser

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"errors"
	"fmt"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
)

// Загрузка, парсинг и раскрытие расписания в одной функции по URI и номеру недели
func (sh *WeekShedule) Download(uri string, week int, uncover bool) error {
	page, err := DownloadShedule(uri, week)
	if err != nil {
		return err
	}
	err = sh.Parse(page, uncover)
	if err != nil {
		return err
	}
	return nil
}

// Загрузка, парсинг и раскрытие расписания в одной функции
// Обязательно наличие IsGroup, SheduleId, Week в объекте
func (sh *WeekShedule) DownloadById(uncover bool) error {
	if sh.SheduleId == 0 {
		return errors.New("schedule id not included")
	}
	if sh.Week == 0 {
		return errors.New("week not included")
	}

	page, err := DownloadSheduleById(sh.SheduleId, sh.IsGroup, sh.Week)
	if err != nil {
		return err
	}
	err = sh.Parse(page, uncover)
	if err != nil {
		return err
	}
	return nil
}

// Раскрытие недельного расписания в список занятий для базы данных и сравнения
func (sh *WeekShedule) UncoverShedule() {
	var lessons []database.Lesson
	for _, line := range sh.Lessons {
		for _, pair := range line {
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
						lessons = append(lessons, l)
					}
				}
			}
		}
	}
	sh.Uncovered = lessons
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

// Получение хэша занятия для быстрого сравнения
func Hash(s database.Lesson) string {
	var b bytes.Buffer
	gob.NewEncoder(&b).Encode(s)
	hash := fmt.Sprintf("%x", md5.Sum(b.Bytes()))
	return hash
}
