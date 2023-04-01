package ssau_parser

import (
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Lesson struct {
	Begin        time.Time
	End          time.Time
	NumInShedule int
	SubLessons   []SubLesson
}

type SubLesson struct {
	Type      string
	Name      string
	Place     string
	TeacherId int64
	GroupId   []int64
	Comment   string
	SubGroup  string
}

type Shedule struct {
	IsGroup   bool
	SheduleId int64
	GroupName string
	SpecName  string
	Week      int
	Lessons   [][]Lesson
}

func GetSheduleInfo(doc *goquery.Document, sh *Shedule) {
	spec := doc.Find(".info-block__description div").First().Text()
	if spec != "" {
		spec = spec[1:]
	}
	sh.SpecName = spec
	sh.GroupName = doc.Find(".info-block__title").First().Text()[1:]

}

// Parse goquery shedule site
func Parse(doc *goquery.Document, isGroup bool, sheduleId int64, week int) (*Shedule, error) {
	var sh Shedule
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

	var lessons [][]SubLesson
	doc.Find(".schedule__item:not(.schedule__head)").Each(func(i int, s *goquery.Selection) {
		sl := ParseSubLesson(s, isGroup, sheduleId)
		lessons = append(lessons, sl)
	})

	var shedule [][]Lesson
	var firstNum int

	for t := 0; t < len(raw_times); t += 2 {
		if t == 0 {
			begin, err := time.Parse(" 15:04 -07", raw_times[t])
			if err != nil {
				return nil, err
			}
			switch begin.Hour() {
			case 8:
				firstNum = 0
			case 9:
				firstNum = 1
			case 11:
				firstNum = 2
			case 13:
				firstNum = 3
			case 15:
				firstNum = 4
			case 17:
				firstNum = 5
			case 18:
				firstNum = 6
			case 20:
				firstNum = 7
			}
		}
		var time_line []Lesson
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
			lesson := Lesson{
				Begin:        begin,
				End:          end,
				NumInShedule: t/2 + firstNum,
				SubLessons:   lessons[idx],
			}
			time_line = append(time_line, lesson)
		}
		shedule = append(shedule, time_line)
	}
	sh.IsGroup = isGroup
	sh.SheduleId = sheduleId
	sh.Week = week
	sh.Lessons = shedule
	return &sh, nil
}

var types = [4]string{"lect", "lab", "pract", "other"}

// Parse shedule item
func ParseSubLesson(s *goquery.Selection, isGroup bool, sheduleId int64) []SubLesson {
	var subs []SubLesson
	s.Find(".schedule__lesson").Each(func(j int, l *goquery.Selection) {
		var sublesson SubLesson

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
