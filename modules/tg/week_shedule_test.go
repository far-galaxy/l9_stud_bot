package tg

import (
	"os"
	"testing"
	"time"

	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/htmlschedule"
)

func TestCreateHTMLShedule(t *testing.T) {
	bot := InitTestBot()

	if _, err := bot.DB.InsertOne(
		database.Teacher{
			TeacherId: 1,
			FirstName: "Иванов",
			LastName:  "Иван Иванович",
			ShortName: "И.И.",
		}); err != nil {
		t.Fatal(err)
	}

	var shedule [][6][]database.Lesson
	var line [6][]database.Lesson
	line[0] = []database.Lesson{
		{
			Type:      "lect",
			Name:      "Занимательная астрология",
			TeacherId: 1,
			Place:     "Дурка",
		},
	}
	line[1] = []database.Lesson{{
		Type: "pract",
		Name: "Тарология",
	},
	}
	line[2] = []database.Lesson{
		{
			Type:      "lect",
			Name:      "АААА",
			TeacherId: 1,
			SubGroup:  1,
			Comment:   "aaa",
		},
		{
			Type:      "pract",
			Name:      "АААА",
			TeacherId: 1,
			SubGroup:  2,
			Comment:   "aaa",
			Place:     "Снова дурка",
		},
	}
	line[3] = []database.Lesson{
		{
			Type:      "lect",
			Name:      "АААА",
			TeacherId: 1,
			SubGroup:  1,
			Comment:   "aaa",
		},
		{
			Type:      "lect",
			Name:      "БББ",
			TeacherId: 1,
			SubGroup:  2,
			Comment:   "aaa",
		},
	}
	shedule = append(shedule, line)

	var dates []time.Time
	var times [][]time.Time
	times = append(times,
		[]time.Time{
			time.Date(2023, 1, 1, 8, 0, 0, 0, time.Local),
			time.Date(2023, 1, 1, 9, 35, 0, 0, time.Local),
		},
	)
	for i := 1; i < 7; i++ {
		dates = append(dates, time.Date(2023, 9, i, 0, 0, 0, 0, time.Local))
	}

	table := api.WeekTable{
		Pairs: shedule,
		Times: times,
		Dates: dates,
	}

	html, _ := htmlschedule.CreateHTMLShedule(
		bot.DB,
		true,
		"Тест",
		table,
	)
	f, _ := os.Create("test_group.html")
	defer f.Close()
	if _, err := f.WriteString(html); err != nil {
		t.Fatal(err)
	}

	html, _ = htmlschedule.CreateHTMLShedule(
		bot.DB,
		false,
		"Тест",
		table,
	)
	f, _ = os.Create("test_staff.html")
	defer f.Close()
	if _, err := f.WriteString(html); err != nil {
		t.Fatal(err)
	}
}
