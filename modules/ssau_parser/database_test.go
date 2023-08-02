package ssau_parser

import (
	"log"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
)

func TestCheckGroupOrTeacher(t *testing.T) {
	db, err := database.Connect("test", "TESTpass1!", "testdb")
	if err != nil {
		log.Println(err)
		return
	}
	// Очистка всех данных для теста
	_, err = db.Where("groupid > 0").Delete(&database.Group{})
	handleError(err)
	_, err = db.Where("teacherid > 0").Delete(&database.Teacher{})
	handleError(err)

	headURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      4,
	}
	err = sh.DownloadById(false)
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
