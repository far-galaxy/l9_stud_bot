package tg

import (
	"os"
	"testing"
	"time"

	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
)

func TestCreateHTMLShedule(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	bot := InitTestBot(files)

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
	}
	shedule = append(shedule, line)

	var dates []time.Time
	var times []ssauparser.Pair
	times = append(times,
		ssauparser.Pair{
			Begin: time.Date(2023, 1, 1, 8, 0, 0, 0, time.Local),
			End:   time.Date(2023, 1, 1, 9, 35, 0, 0, time.Local),
		},
	)
	for i := 1; i < 7; i++ {
		dates = append(dates, time.Date(2023, 9, i, 0, 0, 0, 0, time.Local))
	}

	html := bot.CreateHTMLShedule(
		true,
		"Тест",
		shedule,
		dates,
		times,
	)
	f, _ := os.Create("test.html")
	defer f.Close()
	if _, err := f.WriteString(html); err != nil {
		t.Fatal(err)
	}
}
