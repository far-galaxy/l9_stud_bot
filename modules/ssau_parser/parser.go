package ssau_parser

import (
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Pair struct {
	Begin        time.Time
	End          time.Time
	NumInShedule int
	Lessons      []Lesson
}

type Lesson struct {
	Type      string
	Name      string
	Place     string
	TeacherId int64
	GroupId   []int64
	Comment   string
	SubGroup  string
}

type WeekShedule struct {
	IsGroup   bool
	SheduleId int64
	FullName  string
	SpecName  string
	Week      int
	WeekBegin int
	Lessons   [][]Pair
}

// Получить полный номер группы и название специальности (ФИО и место работы для преподавателей)
func GetSheduleInfo(doc *goquery.Document, sh *WeekShedule) {
	sh.SpecName = strings.TrimSpace(doc.Find(".info-block__description div").First().Text())
	sh.FullName = strings.TrimSpace(doc.Find(".info-block__title").First().Text())

	begin := doc.Find(".info-block__semester div").Last().Text()
	begin = strings.TrimSpace(begin)
	begin = strings.TrimPrefix(begin, "Начало семестра: ")

	startWeekTime, err := time.Parse("02.01.2006", begin)
	if err != nil {
		sh.WeekBegin = 0
	} else {
		_, sh.WeekBegin = startWeekTime.ISOWeek()
	}
}

// Соотнесение часа начала пары с его порядковым номером
var hourMap = map[int]int{8: 0, 9: 1, 11: 2, 13: 3, 15: 4, 17: 5, 18: 6, 20: 7}

// Парсинг страницы с расписанием
func Parse(p Page) (*WeekShedule, error) {
	var sh WeekShedule
	doc := p.Doc
	GetSheduleInfo(doc, &sh)

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
		//sl := ParseLesson(s, p.IsGroup, p.ID)
		sl := []Lesson{
			{},
		}
		lessons = append(lessons, sl)
	})

	var shedule [][]Pair
	var firstNum int

	for t := 0; t < len(raw_times); t += 2 {
		if t == 0 {
			begin, err := time.Parse(" 15:04 -07", raw_times[t])
			if err != nil {
				return nil, err
			}
			firstNum = hourMap[begin.Hour()]
		}

		var time_line []Pair
		for d, date := range raw_dates {
			begin_raw := date + raw_times[t]
			begin, err := time.Parse(" 02.01.2006 15:04 -07", begin_raw)
			if err != nil {
				return nil, err
			}
			end_raw := date + raw_times[t+1]
			end, err := time.Parse(" 02.01.2006 15:04 -07", end_raw)
			if err != nil {
				return nil, err
			}
			idx := (len(raw_dates))*t/2 + d
			lesson := Pair{
				Begin:        begin,
				End:          end,
				NumInShedule: t/2 + firstNum,
				Lessons:      lessons[idx],
			}
			time_line = append(time_line, lesson)
		}
		shedule = append(shedule, time_line)
	}
	sh.IsGroup = p.IsGroup
	sh.SheduleId = p.ID
	sh.Week = p.Week
	sh.Lessons = shedule
	return &sh, nil
}

var types = [4]string{"lect", "lab", "pract", "other"}

// Парсинг занятия
func ParseLesson(s *goquery.Selection, isGroup bool, sheduleId int64) []Lesson {
	var subs []Lesson
	s.Find(".schedule__lesson").Each(func(j int, l *goquery.Selection) {
		var sublesson Lesson

		name := l.Find("div.schedule__discipline").First()
		sublesson.Name = name.Text()[1:]
		l_type := name.AttrOr("class", "lesson-color-type-4")
		t := strings.Split(l_type, " ")
		l_type = t[len(t)-1]
		type_idx, err := strconv.ParseInt(l_type[len(l_type)-1:], 0, 8)
		if err != nil {
			type_idx = 4
		}
		sublesson.Type = types[type_idx-1]

		var teacherId int64
		var groupId []int64

		if isGroup {
			teacher := l.Find(".schedule__teacher a").AttrOr("href", "/rasp?staffId=")
			teacherId, err = strconv.ParseInt(teacher[14:], 0, 64)
			if err != nil {
				teacherId = 0
			}
			groupId = append(groupId, sheduleId)
		} else {
			teacherId = sheduleId
			l.Find("a.schedule__group").Each(func(k int, gr *goquery.Selection) {
				id, err := strconv.ParseInt(gr.AttrOr("href", "/rasp?groupId=")[14:], 0, 64)
				if err != nil {
					teacherId = 0
				}
				groupId = append(groupId, id)
			})
		}
		sublesson.TeacherId = teacherId
		sublesson.GroupId = groupId

		// Я в рот ебал парсить это расписание, потому что у преподов решили номера подгрупп пихать
		// в ссылки на группу, а не в предназначенный для этого элемент
		subgroup := l.Find(".schedule__groups span").First().Text()
		if subgroup == "  " {
			subgroup = ""
		}
		sublesson.SubGroup = subgroup

		place := l.Find("div.schedule__place").First().Text()
		if len(place) > 2 {
			place = place[1:]
		}
		sublesson.Place = place
		sublesson.Comment = l.Find("div.schedule__comment").First().Text()

		subs = append(subs, sublesson)
	})

	return subs
}
