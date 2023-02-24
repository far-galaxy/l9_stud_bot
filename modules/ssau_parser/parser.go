package ssau_parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type RaspList struct {
	Items []RaspItems
}

type RaspItems []struct {
	Id   int64
	Url  string
	Text string
}

func FindInRasp(query string) (RaspItems, error) {
	client := http.Client{}

	req, err := http.NewRequest("GET", "https://ssau.ru/rasp", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	csrf, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists {
		return nil, errors.New("missed csrf")
	}

	parm := url.Values{}
	parm.Add("text", query)
	req, err = http.NewRequest("POST", "https://ssau.ru/rasp/search", strings.NewReader(parm.Encode()))
	if err != nil {
		return nil, err
	}

	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-CSRF-TOKEN", csrf)

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	var list RaspItems
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(body, &list); err != nil {
			return nil, err
		}

	} else {
		return nil, fmt.Errorf("Responce: %s", resp.Status)
	}

	return list, nil
}

type Times struct {
	Begin time.Time
	End   time.Time
}

func Connect(uri string, week int) (*goquery.Document, error) {
	client := http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://ssau.ru%s&selectedWeek=%d", uri, week), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

type Lesson struct {
	Begin time.Time
	End   time.Time
	Name  string
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

	var lessons []string
	doc.Find(".schedule__item:not(.schedule__head)").Each(func(i int, s *goquery.Selection) {
		lesson := s.Text()
		lessons = append(lessons, lesson)
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
				Begin: begin,
				End:   end,
				Name:  lessons[idx],
			}
			time_line = append(time_line, lesson)
		}
		shedule = append(shedule, time_line)
	}
	return &Shedule{SpecName: spec, Lessons: shedule}, nil
}

/*
type Lesson struct {
	Type      string
	Name      string
	Place     string
	TeacherID int64
	Comment   string
}

func parseLesson(l *goquery.Selection) {
	var lesson Lesson
	d, _ := l.Find("div.schedule__discipline").Attr("class")
	t := strings.Split(d, " ")
	lesson.Type = t[len(t)-1]
	lesson.Name = l.Find("div.schedule__discipline").First().Text()
}
*/
