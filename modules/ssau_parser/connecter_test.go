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
}

func pingQuery(query string) {
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
	HeadURL = "http://127.0.0.1:5000"
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
	HeadURL = "http://127.0.0.1:5000/oops/"
	if _, err := DownloadSheduleById(59915001, false, 1); err != nil {
		log.Println(err)
	}
}
