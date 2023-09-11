package ssau_parser

import (
	"strconv"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/exp/slices"
)

// Пара, состоящая из занятий
type Pair struct {
	Begin        time.Time
	End          time.Time
	NumInShedule int
	Lessons      []Lesson
}

// Отдельные занятия внутри пары
type Lesson struct {
	Type      string
	Name      string
	Place     string
	TeacherId []int64
	GroupId   []int64
	Comment   string
	SubGroup  []int
	Hash      []byte
}

// Недельное расписание
type WeekShedule struct {
	IsGroup   bool
	SheduleId int64
	FullName  string
	SpecName  string
	Week      int               // Номер недели в семестре
	WeekBegin int               // Номер недели в году начала семестра
	Lessons   [][]Pair          // Таблица пар в форме недельного расписания
	Uncovered []database.Lesson // Раскрытый список всех занятий для дальнейшей обработки в БД
}

// Получить полный номер группы и название специальности (ФИО и место работы для преподавателей)
func GetSheduleInfo(doc *goquery.Document, sh *WeekShedule) {
	sh.SpecName = strings.TrimSpace(doc.Find(".info-block__description div").First().Text())
	sh.FullName = strings.TrimSpace(doc.Find(".info-block__title").First().Text())
}

// Соотнесение часа начала пары с его порядковым номером
var hourMap = map[int]int{8: 0, 9: 1, 11: 2, 13: 3, 15: 4, 17: 5, 18: 6, 20: 7}

// Парсинг страницы с расписанием
func (sh *WeekShedule) Parse(p Page, uncover bool) error {
	doc := p.Doc
	GetSheduleInfo(doc, sh)

	var raw_dates []string
	doc.Find(".schedule__head-date").Each(func(i int, s *goquery.Selection) {
		sh_date := s.Text()
		raw_dates = append(raw_dates, sh_date)
	})

	var raw_times []string
	doc.Find(".schedule__time-item").Each(func(i int, s *goquery.Selection) {
		sh_time := s.Text() + "+04"
		raw_times = append(raw_times, sh_time)
	})

	var lessons [][]Lesson
	doc.Find(".schedule__item:not(.schedule__head)").Each(func(i int, s *goquery.Selection) {
		sl := ParseLesson(s, p.IsGroup, p.ID)
		lessons = append(lessons, sl)
	})

	var shedule [][]Pair

	for t := 0; t < len(raw_times); t += 2 {
		var time_line []Pair
		for d, date := range raw_dates {
			begin_raw := date + raw_times[t]
			begin, err := time.Parse(" 02.01.2006 15:04 -07", begin_raw)
			if err != nil {
				return err
			}
			end_raw := date + raw_times[t+1]
			end, err := time.Parse(" 02.01.2006 15:04 -07", end_raw)
			if err != nil {
				return err
			}
			idx := (len(raw_dates))*t/2 + d
			lesson := Pair{
				Begin:        begin,
				End:          end,
				NumInShedule: hourMap[begin.Hour()],
				Lessons:      lessons[idx],
			}
			time_line = append(time_line, lesson)
		}
		shedule = append(shedule, time_line)
	}

	if len(shedule) > 2 {
		// Ищем окна
		for y, line := range shedule[1 : len(shedule)-1] {
			for x, pair := range line {
				if len(pair.Lessons) == 0 &&
					len(shedule[y][x].Lessons) != 0 {
					for i := y + 2; i < len(shedule); i++ {
						if len(shedule[i][x].Lessons) != 0 {
							window := Lesson{
								Type: "window",
								Name: "Окно",
							}
							if p.IsGroup {
								window.GroupId = []int64{p.ID}
								window.SubGroup = []int{0}
							} else {
								window.TeacherId = []int64{p.ID}
								window.SubGroup = []int{0}
							}
							shedule[y+1][x].Lessons = []Lesson{window}
							break
						}
					}
				}
			}
		}
		// Оставляем только первую пару военки
		for y, line := range shedule {
			for x, pair := range line {
				if len(pair.Lessons) > 0 && pair.Lessons[0].Type == "mil" {
					for i := y + 1; i < len(shedule); i++ {
						if len(shedule[i][x].Lessons) > 0 && shedule[i][x].Lessons[0].Type == "mil" {
							shedule[i][x].Lessons = []Lesson{}
						}
					}
				}
			}
		}
	}
	sh.IsGroup = p.IsGroup
	sh.SheduleId = p.ID
	sh.Week = p.Week
	sh.Lessons = shedule

	if uncover {
		sh.UncoverShedule()
	}
	return nil
}

var types = []string{"lect", "lab", "pract", "other", "exam", "cons", "kurs"}

// Парсинг занятия
func ParseLesson(s *goquery.Selection, isGroup bool, sheduleId int64) []Lesson {
	var lessons []Lesson
	s.Find(".schedule__lesson").Each(func(j int, l *goquery.Selection) {
		var lesson Lesson

		name := l.Find("div.schedule__discipline").First()
		lesson.Name = strings.TrimSpace(name.Text())
		if strings.ToLower(lesson.Name) == "военная подготовка" {
			lesson.Type = "mil"
		} else {
			l_type := name.AttrOr("class", "lesson-color-type-4")
			t := strings.Split(l_type, " ")
			l_type = t[len(t)-1]
			type_idx, err := strconv.ParseInt(l_type[len(l_type)-1:], 0, 8)
			if err != nil {
				type_idx = 4
			}
			lesson.Type = types[type_idx-1]
		}

		var teacherId []int64
		var groupId []int64

		l.Find(".schedule__teacher a").Each(func(k int, gr *goquery.Selection) {
			id, err := strconv.ParseInt(gr.AttrOr("href", "/rasp?staffId=")[14:], 0, 64)
			if err != nil {
				return
			}
			teacherId = append(teacherId, id)
		})

		l.Find("a.schedule__group").Each(func(k int, gr *goquery.Selection) {
			id, err := strconv.ParseInt(gr.AttrOr("href", "/rasp?groupId=")[14:], 0, 64)
			if err != nil {
				return
			}
			groupId = append(groupId, id)

			// Вытягиваем подгруппу из преподавательского расписания
			if !isGroup {
				group := gr.First().Text()
				if idx := strings.Index(group, "("); idx != -1 {
					if endIdx := strings.Index(group[idx:], ")"); endIdx != -1 {
						if sub, err := strconv.Atoi(group[idx+1 : idx+endIdx]); err == nil {
							lesson.SubGroup = append(lesson.SubGroup, sub)
						}
					}
				} else {
					lesson.SubGroup = append(lesson.SubGroup, 0)
				}
			}
		})

		// Добавляем, собственно, группу или преподавателя настоящего расписания
		if isGroup {
			if !slices.Contains(groupId, sheduleId) {
				groupId = append(groupId, sheduleId)
			}
		} else {
			teacherId = append(teacherId, sheduleId)
		}

		lesson.TeacherId = teacherId
		lesson.GroupId = groupId

		if isGroup && len(groupId) == 1 {
			subgroup := strings.TrimSpace(l.Find(".schedule__groups span").First().Text())
			if len(subgroup) != 0 {
				subgroup = strings.Split(subgroup, ":")[1]
				subgroupNum, _ := strconv.Atoi(strings.TrimSpace(subgroup))
				lesson.SubGroup = append(lesson.SubGroup, subgroupNum)
			} else {
				lesson.SubGroup = append(lesson.SubGroup, 0)
			}
		} else if isGroup && len(groupId) > 1 {
			for range groupId {
				lesson.SubGroup = append(lesson.SubGroup, 0)
			}
		}

		place := l.Find("div.schedule__place").First().Text()
		place = strings.TrimSpace(place)
		lesson.Place = place
		lesson.Comment = l.Find("div.schedule__comment").First().Text()

		lessons = append(lessons, lesson)
	})

	return lessons
}
