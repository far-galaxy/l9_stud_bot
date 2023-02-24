package ssau_parser

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Lesson struct {
	Begin      time.Time
	End        time.Time
	SubLessons []SubLesson
}

type SubLesson struct {
	Type      string
	Name      string
	Place     string
	TeacherID int64
	Comment   string
	SubGroup  string
}

type Shedule struct {
	SpecName string
	Week     int
	Lessons  [][]Lesson
}

func Parse(doc *goquery.Document) (*Shedule, error) {
	spec := doc.Find(".info-block__description div").First().Text()[1:]
	log.Println(spec)

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
		sl := ParseSubLesson(s)
		lessons = append(lessons, sl)
	})

	var shedule [][]Lesson

	for t := 0; t < len(raw_times); t += 2 {
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
			idx := (len(raw_times)-2)*t/2 + d
			lesson := Lesson{
				Begin:      begin,
				End:        end,
				SubLessons: lessons[idx],
			}
			time_line = append(time_line, lesson)
		}
		shedule = append(shedule, time_line)
	}
	return &Shedule{SpecName: spec, Lessons: shedule}, nil
}

var types = [4]string{"lect", "lab", "pract", "other"}

func ParseSubLesson(s *goquery.Selection) []SubLesson {
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

		teacher := l.Find(".schedule__teacher a").AttrOr("href", "/rasp?staffId=")
		teacherId, err := strconv.ParseInt(teacher[14:], 0, 64)
		if err != nil {
			teacherId = 0
		}
		sublesson.TeacherID = teacherId

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
