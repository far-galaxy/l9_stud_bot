package notify

import (
	"log"
	"os"
	"testing"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
)

var TestDB = database.DB{
	User:   "test",
	Pass:   "TESTpass1!",
	Schema: "testdb",
}

func TestCheckNext(t *testing.T) {
	files := database.OpenLogs()
	defer files.CloseAll()
	db, err := database.Connect(TestDB, files.DBLogFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Where("lessonid > 0").Delete(&database.Lesson{}); err != nil {
		t.Fatal(err)
	}
	lessons := []database.Lesson{
		{
			Begin:        time.Date(2032, 2, 1, 8, 0, 0, 0, time.Local),
			End:          time.Date(2032, 2, 1, 9, 35, 0, 0, time.Local),
			NumInShedule: 1,
			GroupId:      1,
		},
		{
			Begin:        time.Date(2032, 2, 1, 9, 45, 0, 0, time.Local),
			End:          time.Date(2032, 2, 1, 11, 20, 0, 0, time.Local),
			NumInShedule: 2,
			GroupId:      1,
		},
		{
			Begin:        time.Date(2032, 2, 2, 8, 0, 0, 0, time.Local),
			End:          time.Date(2032, 2, 2, 9, 35, 0, 0, time.Local),
			NumInShedule: 1,
			GroupId:      1,
		},
		{
			Begin:        time.Date(2032, 2, 9, 8, 0, 0, 0, time.Local),
			End:          time.Date(2032, 2, 9, 9, 35, 0, 0, time.Local),
			NumInShedule: 1,
			GroupId:      1,
		},
	}
	for _, l := range lessons {
		l.Hash = ssau_parser.Hash(l)
	}
	if _, err := db.Insert(&lessons); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2032, 2, 1, 9, 35, 20, 10, time.Local)
	if _, err := CheckNext(db, now); err != nil {
		t.Fatal(err)
	}
}

func TestFirstMailing(t *testing.T) {
	if err := tg.CheckEnv(); err != nil {
		log.Fatal(err)
	}
	logs := database.OpenLogs()
	defer logs.CloseAll()
	bot, err := tg.InitBot(
		logs,
		database.DB{
			User:   os.Getenv("MYSQL_USER"),
			Pass:   os.Getenv("MYSQL_PASS"),
			Schema: os.Getenv("MYSQL_DB"),
		},
		os.Getenv("TELEGRAM_APITOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	now, _ := time.Parse("2006-01-02 15:04 -07", "2023-02-06 07:15 +04")
	FirstMailing(bot, now)
}
