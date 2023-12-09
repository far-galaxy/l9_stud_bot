package api

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/icza/gox/timex"
	"stud.l9labs.ru/bot/modules/database"
	"xorm.io/xorm"
)

type WeekTable struct {
	Pairs [][6][]database.Lesson // Пары, разложенные по полочкам
	Times [][]time.Time          // Времена начала и конца пары соответственно в течение дня (верт. ось)
	Dates []time.Time            // Даты в течение недели (гориз. ось)
}

var days = [6]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// Получить расписание в табличном виде для конвертации в HTML
//
// В параметр week указывается неделя года (!)
func GetWeekOrdered(db *xorm.Engine, shedule database.Schedule, week int) (WeekTable, error) {
	var weekTable WeekTable
	lessons, err := GetWeekLessons(db, shedule, week)
	if err != nil {
		return weekTable, err
	}
	// Глядим на неделю вперёд
	if len(lessons) == 0 {
		next, err := GetWeekLessons(db, shedule, week+1)
		if err != nil {
			return weekTable, err
		}
		if len(next) > 0 {
			lessons = next
			week++
		} else {
			return weekTable, fmt.Errorf("no lessons: %d, week %d", shedule.ScheduleID, week)
		}
	}

	begins := make(map[time.Time]bool)
	ends := make(map[time.Time]bool)
	height := 0
	minDay := lessons[0].NumInShedule
	for _, lesson := range lessons {
		t := lesson.Begin
		begin := time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, t.Location())
		begins[begin] = true

		e := lesson.End
		end := time.Date(2000, 1, 1, e.Hour(), e.Minute(), 0, 0, e.Location())
		ends[end] = true

		if lesson.NumInShedule > height {
			height = lesson.NumInShedule
		} else if lesson.NumInShedule < minDay {
			minDay = lesson.NumInShedule
		}
	}
	// Собираем всё время в одну кучу и сортируем
	var times [][]time.Time
	var beginsSlice []time.Time
	var endsSlice []time.Time
	for b := range begins {
		beginsSlice = append(beginsSlice, b)
	}
	for e := range ends {
		endsSlice = append(endsSlice, e)
	}
	sort.Slice(beginsSlice, func(i, j int) bool {
		return beginsSlice[i].Before(beginsSlice[j])
	})
	sort.Slice(endsSlice, func(i, j int) bool {
		return endsSlice[i].Before(endsSlice[j])
	})
	for i := range beginsSlice {
		sh := []time.Time{beginsSlice[i], endsSlice[i]}
		times = append(times, sh)
	}

	// Создаём даты на указанную неделю
	var dates []time.Time
	weekBegin := timex.WeekStart(lessons[0].Begin.Year(), week)
	for i := range days {
		dates = append(dates, weekBegin.Add(time.Hour*time.Duration(24*i)))
	}

	// Группируем занятия по парам и распределяем в таблицу
	table := make([][6][]database.Lesson, height-minDay+1)
	pairs := GroupPairs(lessons)
	for _, p := range pairs {
		day := int(math.Floor(p[0].Begin.Sub(weekBegin).Hours() / 24))
		table[p[0].NumInShedule-minDay][day] = p
	}

	// Проверяем пустые строки и эпирическим путём подбираем для них время (или не подбираем вовсе)
	for y, line := range table {
		count := 0
		for _, l := range line {
			count += len(l)
		}
		if count == 0 {
			nilPair := make([]time.Time, 2)
			if y == len(table) {
				times = append(times, nilPair)
			} else {
				times = append(times[:y+1], times[y:]...)
				times[y] = nilPair
			}
		}
	}
	weekTable.Pairs = table
	weekTable.Times = times
	weekTable.Dates = dates

	return weekTable, nil
}

// Проверка, не закончились ли пары на этой неделе
//
// При week == -1 неделя определяется автоматически
func CheckWeek(db *xorm.Engine, now time.Time, week *int, schedule database.Schedule) (bool, error) {
	if *week == -1 || *week == 0 {
		_, nowWeek := now.ISOWeek()
		*week = nowWeek
		lesson, err := GetNearLesson(db, schedule, now)
		if err != nil {
			return false, err
		}
		if lesson.LessonId != 0 {
			_, lessonWeek := lesson.Begin.ISOWeek()
			if lessonWeek > nowWeek {
				*week++

				return true, nil
			}

			return false, nil
		}
	}

	return false, nil
}
