package ssau_parser

import (
	"log"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"xorm.io/xorm"
)

var TestDB = database.DB{
	User:   "test",
	Pass:   "TESTpass1!",
	Schema: "testdb",
}

func prepareDB() *xorm.Engine {
	db, err := database.Connect(TestDB)
	if err != nil {
		log.Println(err)
		return nil
	}
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

	headURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      4,
	}
	err := sh.DownloadById(false)
	handleError(err)
	err = checkGroupOrTeacher(db, sh)
	handleError(err)
	// Повторяем на предмет наличия
	err = checkGroupOrTeacher(db, sh)
	handleError(err)

	// Проверяем преподавателя
	sh = WeekShedule{
		SheduleId: 62806001,
		IsGroup:   false,
		Week:      4,
	}
	err = sh.DownloadById(false)
	handleError(err)
	err = checkGroupOrTeacher(db, sh)
	handleError(err)
}

func TestUpdateSchedule(t *testing.T) {
	db := prepareDB()
	headURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      4,
	}
	err := sh.DownloadById(true)
	handleError(err)
	_, _, err = UpdateSchedule(db, sh)
	handleError(err)

	sh.Week = 8
	err = sh.DownloadById(true)
	handleError(err)
	_, _, err = UpdateSchedule(db, sh)
	handleError(err)
}