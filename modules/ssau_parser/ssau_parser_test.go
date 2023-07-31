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
