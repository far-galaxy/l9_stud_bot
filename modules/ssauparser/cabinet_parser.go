package ssauparser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
)

type LessonJSON struct {
	ID   int `json:"id"`
	Type struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"type"`
	Weeks []struct {
		Week     int `json:"week"`
		Building struct {
			Name string `json:"name"`
		} `json:"building"`
		Room struct {
			Name string `json:"name"`
		} `json:"room"`
		IsOnline int `json:"isOnline"`
	} `json:"weeks"`
	Groups []struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Subgroup int64  `json:"subgroup"`
	} `json:"groups"`
	Teachers []struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		State string `json:"state"`
	} `json:"teachers"`
	Time struct {
		NumInSchedule int    `json:"id"`
		Name          string `json:"name"`
		Begin         string `json:"beginTime"`
		End           string `json:"endTime"`
	} `json:"time"`
	Discipline struct {
		Name string `json:"name"`
	} `json:"discipline"`
	Weekday struct {
		Num    int    `json:"id"` // 1 - пн
		Abbrev string `json:"abbrev"`
	} `json:"weekday"`
	Comment      string `json:"comment"`
	WeeklyDetail bool   `json:"weeklyDetail"`
	Conference   struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	} `json:"conference"`
}

type Data struct {
	Lessons    []LessonJSON  `json:"lessons"`
	IetLessons []interface{} `json:"ietLessons"`
	Sfc        []interface{} `json:"sfc"`
	Year       struct {
		StartDate string `json:"startDate"`
	} `json:"currentYear"`
}

// Запрос на расписание из ЛК
type Query struct {
	YearID       int64
	Week         int64
	GroupID      int64
	SessionToken string
}

// Загрузка расписания из ЛК
func LoadJSON(query Query) (Data, error) {
	var data Data
	client := http.Client{}

	// Теперь можно обращаться к подобию API
	req, err := http.NewRequest(
		"GET",
		"https://cabinet.ssau.ru/api/timetable/get-timetable",
		nil,
	)
	if err != nil {
		return data, err
	}

	q := req.URL.Query()
	q.Add("yearId", fmt.Sprintf("%d", query.YearID))
	q.Add("week", fmt.Sprintf("%d", query.Week))
	q.Add("userType", "student")
	q.Add("groupId", fmt.Sprintf("%d", query.GroupID))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("Accept", "application/json")
	c := http.Cookie{
		Name:  "laravel_session",
		Value: query.SessionToken,
	}
	req.AddCookie(&c)

	resp, err := client.Do(req)
	if err != nil {
		return data, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return data, err
		}

		if err := json.Unmarshal(body, &data); err != nil {
			return data, err
		}

	} else {
		return data, fmt.Errorf("responce %s: %s", resp.Status, req.URL)
	}

	return data, nil
}

// Парсинг JSON расписания
func ParseData(q Query, d Data) ([]database.Lesson, error) {
	var lessons []database.Lesson
	for _, i := range d.Lessons {
		l := database.Lesson{
			NumInShedule: i.Time.NumInSchedule - 1,
			Type:         types[i.Type.ID-1],
			Name:         i.Discipline.Name,
			Comment:      i.Comment,
		}
		start, err := time.Parse("2006-01-02", d.Year.StartDate)
		if err != nil {
			return lessons, err
		}
		begin, err := time.ParseDuration(
			strings.ReplaceAll(i.Time.Begin, ":", "h") + "m",
		)
		if err != nil {
			return lessons, err
		}
		end, err := time.ParseDuration(
			strings.ReplaceAll(i.Time.End, ":", "h") + "m",
		)
		if err != nil {
			return lessons, err
		}
		year, firstWeek := start.ISOWeek()
		dates, _ := api.GetWeekDates(year, firstWeek+int(q.Week))

		for _, t := range i.Teachers {
			l.TeacherId = int64(t.ID)

			// Удаляем коллективных преподавателей
			if l.TeacherId < 10000 {
				l.TeacherId = 0
			}
			for _, g := range i.Groups {
				if g.ID != q.GroupID {
					continue
				}

				l.GroupId = g.ID
				l.SubGroup = g.Subgroup
				for _, w := range i.Weeks {
					if w.Week != int(q.Week) {
						continue
					}
					l.Begin = dates[i.Weekday.Num-1].Add(begin)
					l.End = dates[i.Weekday.Num-1].Add(end)
					l.Place = fmt.Sprintf("%s-%s", w.Room.Name, w.Building.Name)
					lessons = append(lessons, l)
				}
			}
		}
	}

	return lessons, nil
}
