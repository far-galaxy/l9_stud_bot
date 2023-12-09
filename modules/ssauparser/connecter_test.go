package ssauparser

import (
	"log"
	"testing"
)

const testURL = "http://127.0.0.1:5000"

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
	testURL,
	testURL,
}

// Вывод некритических ошибок тестирования в консоль
func handleError(err error) {
	if err != nil {
		log.Println(err)
	}
}

func TestSearchInRasp(t *testing.T) {
	// Проверка запросов
	for _, query := range queries {
		pingQuery(query)
	}
	// Проверка ошибок на стороне сайта
	for _, url := range urls {
		HeadURL = url
		pingQuery(queries[0])
	}
	t.Log("ok")
}

func pingQuery(query string) {
	if list, err := SearchInRasp(query); err != nil {
		log.Println(err)
	} else {
		log.Println(query, list)
	}
}

var groupURI = []string{
	"/rasp?groupId=100000000",
	"/rasp?staffId=1",
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
	HeadURL = testURL
	for _, uri := range groupURI {
		for _, week := range weeks {
			if _, err := DownloadShedule(uri, week); err != nil {
				log.Println(err)
			}
		}
	}
	if _, err := DownloadSheduleByID(100000000, true, 1); err != nil {
		log.Println(err)
	}
	if _, err := DownloadSheduleByID(1, false, 1); err != nil {
		log.Println(err)
	}
	HeadURL = "http://127.0.0.1:5000/oops/"
	if _, err := DownloadSheduleByID(100000000, false, 1); err != nil {
		log.Println(err)
	}
	t.Log("ok")
}
