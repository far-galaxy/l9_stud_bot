package ssauparser

import (
	"log"
	"testing"

	"stud.l9labs.ru/bot/modules/database"
	"xorm.io/xorm"
)

var TestDB = database.DB{
	User:   "test",
	Pass:   "TESTpass1!",
	Schema: "testdb",
}

func prepareDB() *xorm.Engine {
	logs := database.OpenLogs()
	db, err := database.Connect(TestDB, logs.DBLogFile)
	if err != nil {
		log.Println(err)

		return nil
	}
	defer logs.CloseAll()
	// Очистка всех данных для теста
	_, err = db.Where("groupid > 0").Delete(&database.Group{})
	handleError(err)
	_, err = db.Where("teacherid > 0").Delete(&database.Teacher{})
	handleError(err)
	_, err = db.Where("lessonid >= 0").Delete(&database.Lesson{})
	handleError(err)

	return db
}

func TestCheckGroupOrTeacher(t *testing.T) {
	db := prepareDB()

	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleID: 123456789,
		IsGroup:   true,
		Week:      6,
	}
	err := sh.DownloadByID(false)
	handleError(err)
	_, err = CheckGroupOrTeacher(db, sh)
	handleError(err)
	// Повторяем на предмет наличия
	_, err = CheckGroupOrTeacher(db, sh)
	handleError(err)

	// Проверяем преподавателя
	sh = WeekShedule{
		SheduleID: 5,
		IsGroup:   false,
		Week:      4,
	}
	err = sh.DownloadByID(false)
	handleError(err)
	_, err = CheckGroupOrTeacher(db, sh)
	handleError(err)
	t.Log("ok")
}

func TestUpdateSchedule(t *testing.T) {
	db := prepareDB()
	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleID: 123456789,
		IsGroup:   true,
		Week:      6,
	}
	err := sh.DownloadByID(true)
	handleError(err)
	_, _, err = UpdateSchedule(db, sh)
	handleError(err)

	sh.Week = 7
	err = sh.DownloadByID(true)
	handleError(err)
	_, _, err = UpdateSchedule(db, sh)
	handleError(err)

	// Проверяем преподавателя
	sh = WeekShedule{
		SheduleID: 5,
		IsGroup:   false,
		Week:      4,
	}
	err = sh.DownloadByID(true)
	handleError(err)
	_, _, err = UpdateSchedule(db, sh)
	handleError(err)
	t.Log("ok")
}
