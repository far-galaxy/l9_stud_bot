package ssauparser

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"stud.l9labs.ru/bot/modules/database"
	"xorm.io/xorm"
)

func prepareDB() *xorm.Engine {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
	DB := database.DB{
		User:   os.Getenv("L9_MYSQL_USER"),
		Pass:   os.Getenv("L9_MYSQL_PASS"),
		Schema: os.Getenv("L9_MYSQL_DATABASE"),
		Host:   os.Getenv("L9_MYSQL_HOST"),
		Port:   os.Getenv("L9_MYSQL_PORT"),
	}
	db, err := database.Connect(DB, database.InitLog("sql", time.Hour*24*14))
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

	HeadURL = testURL
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
	HeadURL = testURL
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
