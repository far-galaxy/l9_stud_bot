package ssauparser

import (
	"bytes"
	"crypto/md5" // #nosec G501
	"encoding/gob"
	"errors"
	"fmt"

	"stud.l9labs.ru/bot/modules/database"
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
// Обязательно наличие IsGroup, SheduleID, Week в объекте
func (sh *WeekShedule) DownloadByID(uncover bool) error {
	if sh.SheduleID == 0 {
		return errors.New("schedule id not included")
	}
	if sh.Week == 0 {
		return errors.New("week not included")
	}

	page, err := DownloadSheduleByID(sh.SheduleID, sh.IsGroup, sh.Week)
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
				for i, gID := range lesson.GroupID {
					if len(lesson.TeacherID) == 0 {
						lesson.TeacherID = append(lesson.TeacherID, 0)
					}
					for _, tID := range lesson.TeacherID {
						l := database.Lesson{
							NumInShedule: pair.NumInShedule,
							Type:         lesson.Type,
							Name:         lesson.Name,
							Begin:        pair.Begin,
							End:          pair.End,
							StaffID:      tID,
							GroupID:      gID,
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
	hashes := make(map[string]bool)

	// Создаем карту хешей из второго списка
	for _, lesson := range dzwa {
		hashes[lesson.Hash] = true
	}

	var diff []database.Lesson

	// Проверяем каждый элемент из первого списка
	for _, lesson := range jeden {
		if _, found := hashes[lesson.Hash]; !found {
			diff = append(diff, lesson)
		}
	}

	return diff
}

// Получение хэша занятия для быстрого сравнения
func Hash(s database.Lesson) string {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(s); err != nil {
		return "0x0"
	}

	return fmt.Sprintf("%x", md5.Sum(b.Bytes())) // #nosec G401
}
