package ssau_parser

import (
	"log"
	"testing"
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
	page, err := DownloadSheduleById(802440189, true, 3)
	if err != nil {
		log.Println(err)
		return
	}
	_, err = Parse(page)
	if err != nil {
		log.Println(err)
	}

	// Ошибки в скелете расписания
	for i := 1; i < 6; i++ {
		page, err := DownloadSheduleById(123, true, i)
		if err != nil {
			log.Println(err)
			return
		}
		_, err = Parse(page)
		if err != nil {
			log.Println(err)
		}
	}

	// Ошибки внутри пар
	for i := 2; i < 3; i++ {
		page, err := DownloadSheduleById(62806001, false, i)
		if err != nil {
			log.Println(err)
			return
		}
		sh, err := Parse(page)
		if err != nil {
			log.Println(err)
		}
		log.Println(sh.FullName)
	}
}

func TestSheduleCompare(t *testing.T) {
	headURL = "http://127.0.0.1:5000"
	page, err := DownloadSheduleById(802440189, true, 4)
	if err != nil {
		log.Println(err)
		return
	}
	sh, err := Parse(page)
	if err != nil {
		log.Println(err)
	}
	lessons := UncoverShedule(*sh)

	page, err = DownloadSheduleById(802440189, true, 8)
	if err != nil {
		log.Println(err)
		return
	}
	sh, err = Parse(page)
	if err != nil {
		log.Println(err)
	}
	new_lessons := UncoverShedule(*sh)

	add, del := Compare(new_lessons, lessons)
	log.Println(add, del)
}
