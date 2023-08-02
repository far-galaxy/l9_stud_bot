package ssau_parser

import (
	"log"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
)

var queries = []string{
	"2305",
	"2305-240502D",
	"235",
	"Балякин",
	"Балялякин",
}

var urls = []string{
	"aaa",
	"https://sasau.ru",
	"https://l9labs.ru",
	"http://127.0.0.1:5000",
	"http://127.0.0.1:5000",
}

// Вывод некритических ошибок тестирования в консоль
func handleError(err error) {
	if err != nil {
		log.Println(err)
	}
}

// TODO: выдумать и прописать упоротые тесты для всего
func TestSearchInRasp(t *testing.T) {
	// Проверка запросов
	for _, query := range queries {
		pingQuery(query, t)
	}
	// Проверка ошибок на стороне сайта
	for _, url := range urls {
		headURL = url
		pingQuery(queries[0], t)
	}
}

func pingQuery(query string, t *testing.T) {
	if list, err := SearchInRasp(query); err != nil {
		log.Println(err)
	} else {
		log.Println(query, list)
	}
}

var groupUri = []string{
	"/rasp?groupId=530996168",
	"/rasp?staffId=59915001",
	"/aaa",
	"/aaaaaaaaaaaaaa",
	"/rasp?groupId=123",
}
var weeks = []int{
	1,
	2,
	100,
}

func TestDownloadShedule(t *testing.T) {
	// headURL = "https://ssau.ru"
	headURL = "http://127.0.0.1:5000"
	for _, uri := range groupUri {
		for _, week := range weeks {
			if _, err := DownloadShedule(uri, week); err != nil {
				log.Println(err)
			}
		}
	}
	if _, err := DownloadSheduleById(530996168, true, 1); err != nil {
		log.Println(err)
	}
	if _, err := DownloadSheduleById(59915001, false, 1); err != nil {
		log.Println(err)
	}
}

func TestParse(t *testing.T) {
	headURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      3,
	}
	err := sh.DownloadById(false)
	handleError(err)

	// Ошибки в скелете расписания
	for i := 1; i < 6; i++ {
		sh := WeekShedule{
			SheduleId: 123,
			IsGroup:   true,
			Week:      i,
		}
		err = sh.DownloadById(false)
		handleError(err)
	}

	// Ошибки внутри пар
	for i := 2; i < 3; i++ {
		sh := WeekShedule{
			SheduleId: 62806001,
			IsGroup:   false,
			Week:      i,
		}
		err = sh.DownloadById(false)
		handleError(err)
		log.Println(sh.FullName)
	}
}

func TestSheduleCompare(t *testing.T) {
	headURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      4,
	}
	err := sh.DownloadById(true)
	handleError(err)

	new_sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      8,
	}
	err = new_sh.DownloadById(true)
	handleError(err)

	add, del := Compare(new_sh.Uncovered, sh.Uncovered)
	log.Println(add, del)
}

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
}
