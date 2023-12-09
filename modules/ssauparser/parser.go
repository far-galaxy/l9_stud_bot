package ssauparser

import (
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/exp/slices"
	"stud.l9labs.ru/bot/modules/database"
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
	Type      database.Kind
	Name      string
	Place     string
	TeacherID []int64
	GroupID   []int64
	Comment   string
	SubGroup  []int
	Hash      []byte
}

// Недельное расписание
type WeekShedule struct {
	IsGroup   bool
	SheduleID int64
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

	var rawDates []string
	doc.Find(".schedule__head-date").Each(func(i int, s *goquery.Selection) {
		shDate := s.Text()
		rawDates = append(rawDates, shDate)
	})

	var rawTimes []string
	doc.Find(".schedule__time-item").Each(func(i int, s *goquery.Selection) {
		shTime := s.Text() + "+04"
		rawTimes = append(rawTimes, shTime)
	})

	var lessons [][]Lesson
	doc.Find(".schedule__item:not(.schedule__head)").Each(func(i int, s *goquery.Selection) {
		sl := ParseLesson(s, p.IsGroup, p.ID)
		lessons = append(lessons, sl)
	})

	shedule, err := createPairArray(rawTimes, rawDates, lessons)
	if err != nil {
		return err
	}

	if len(shedule) > 2 {
		shedule = p.findWindows(shedule)
		shedule = clearMilitary(shedule)
	}
	sh.IsGroup = p.IsGroup
	sh.SheduleID = p.ID
	sh.Week = p.Week
	sh.Lessons = shedule

	if uncover {
		sh.UncoverShedule()
	}

	return nil
}

// Оставить только первую пару военки
func clearMilitary(shedule [][]Pair) [][]Pair {
	for y, line := range shedule {
		for x, pair := range line {
			if len(pair.Lessons) > 0 && pair.Lessons[0].Type == database.Military {
				//dayStr := pair.Begin.Format("2006-01-02")
				//shedule[y][x].Begin, _ = time.Parse("2006-01-02 15:04 -07", dayStr+" 08:30 +04")
				//shedule[y][x].End, _ = time.Parse("2006-01-02 15:04 -07", dayStr+" 17:20 +04")
				for i := y + 1; i < len(shedule); i++ {
					if len(shedule[i][x].Lessons) > 0 && shedule[i][x].Lessons[0].Type == database.Military {
						shedule[i][x].Lessons = []Lesson{}
					}
				}
			}
		}
	}

	return shedule
}

// Поиск окон
func (p *Page) findWindows(shedule [][]Pair) [][]Pair {
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
							window.GroupID = []int64{p.ID}
							window.SubGroup = []int{0}
						} else {
							window.TeacherID = []int64{p.ID}
							window.SubGroup = []int{0}
						}
						shedule[y+1][x].Lessons = []Lesson{window}

						break
					}
				}
			}
		}
	}

	return shedule
}

func createPairArray(rawTimes []string, rawDates []string, lessons [][]Lesson) ([][]Pair, error) {
	var shedule [][]Pair
	for t := 0; t < len(rawTimes); t += 2 {
		var timeLine []Pair
		for d, date := range rawDates {
			beginRaw := date + rawTimes[t]
			begin, err := time.Parse(" 02.01.2006 15:04 -07", beginRaw)
			if err != nil {
				return nil, err
			}
			endRaw := date + rawTimes[t+1]
			end, err := time.Parse(" 02.01.2006 15:04 -07", endRaw)
			if err != nil {
				return nil, err
			}
			idx := (len(rawDates))*t/2 + d
			lesson := Pair{
				Begin:        begin,
				End:          end,
				NumInShedule: hourMap[begin.Hour()],
				Lessons:      lessons[idx],
			}
			timeLine = append(timeLine, lesson)
		}
		shedule = append(shedule, timeLine)
	}

	return shedule, nil
}

var types = []database.Kind{
	database.Lection, database.Lab, database.Practice, database.Other,
	database.Exam, database.Consult, database.CourseWork,
	database.Test,
}

// Парсинг занятия
func ParseLesson(s *goquery.Selection, isGroup bool, sheduleID int64) []Lesson {
	var lessons []Lesson
	s.Find(".schedule__lesson").Each(func(j int, l *goquery.Selection) {
		var lesson Lesson

		name := l.Find("div.schedule__discipline").First()
		lesson.Name = strings.TrimSpace(name.Text())
		lesson.parseType(name)

		var teacherID []int64
		var groupID []int64

		l.Find(".schedule__teacher a").Each(func(k int, gr *goquery.Selection) {
			id, err := strconv.ParseInt(gr.AttrOr("href", "/rasp?staffId=")[14:], 0, 64)
			if err != nil {
				return
			}
			teacherID = append(teacherID, id)
		})

		l.Find("a.schedule__group").Each(func(k int, gr *goquery.Selection) {
			id, err := strconv.ParseInt(gr.AttrOr("href", "/rasp?groupId=")[14:], 0, 64)
			if err != nil {
				return
			}
			groupID = append(groupID, id)
			lesson.parseTeacherSubgroups(isGroup, gr)
		})

		// Добавляем, собственно, группу или преподавателя настоящего расписания
		if isGroup {
			if !slices.Contains(groupID, sheduleID) {
				groupID = append(groupID, sheduleID)
			}
		} else {
			teacherID = append(teacherID, sheduleID)
		}

		lesson.TeacherID = teacherID
		lesson.GroupID = groupID

		lesson.parseSubgroups(isGroup, groupID, l)

		place := l.Find("div.schedule__place").First().Text()
		place = strings.TrimSpace(place)
		lesson.Place = place
		lesson.Comment = l.Find("div.schedule__comment").First().Text()

		lessons = append(lessons, lesson)
	})

	return lessons
}

// Вытягиваем подгруппу из преподавательского расписания
func (lesson *Lesson) parseTeacherSubgroups(isGroup bool, gr *goquery.Selection) {
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
}

// Поиск подгрупп
func (lesson *Lesson) parseSubgroups(isGroup bool, groupID []int64, l *goquery.Selection) {
	if isGroup && len(groupID) == 1 {
		subgroup := strings.TrimSpace(l.Find(".schedule__groups span").First().Text())
		if len(subgroup) != 0 {
			subgroup = strings.Split(subgroup, ":")[1]
			subgroupNum, _ := strconv.Atoi(strings.TrimSpace(subgroup))
			lesson.SubGroup = append(lesson.SubGroup, subgroupNum)
		} else {
			lesson.SubGroup = append(lesson.SubGroup, 0)
		}
	} else if isGroup && len(groupID) > 1 {
		for range groupID {
			lesson.SubGroup = append(lesson.SubGroup, 0)
		}
	}
}

// Определение типа занятия
func (lesson *Lesson) parseType(name *goquery.Selection) {
	if strings.ToLower(lesson.Name) == "военная подготовка" {
		lesson.Type = database.Military

		return
	}
	lType := name.AttrOr("class", "lesson-color-type-4")
	t := strings.Split(lType, " ")
	lType = t[len(t)-1]
	typeIdx, err := strconv.ParseInt(lType[len(lType)-1:], 0, 8)
	if err != nil {
		lesson.Type = database.Other

		return
	}
	if int(typeIdx) > len(types) {
		lesson.Type = database.Unknown

		return
	}
	lesson.Type = types[typeIdx-1]
}
