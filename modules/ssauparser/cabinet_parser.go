package ssauparser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		BeginTime     string `json:"beginTime"`
		EndTime       string `json:"endTime"`
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
}

// Запрос на расписание из ЛК
type Query struct {
	YearID  int64
	Week    int64
	GroupID int64
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
		Value: "",
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
