package ssau_parser

import (
	"log"
	"os"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"github.com/joho/godotenv"
)

// TODO: выдумать и прописать упоротые тесты для всего
func TestFindInRasp(t *testing.T) {
	list, err := SearchInRasp("2305")
	if err != nil {
		t.Error(err)
	}
	log.Println(list)
}

func TestConnect(t *testing.T) {
	list, err := SearchInRasp("2305")
	if err != nil {
		t.Error(err)
	}
	uri := list[0].Url
	_, _, _, err = DownloadShedule(uri, 3)
	if err != nil {
		t.Error(err)
	}
}

func TestParse(t *testing.T) {
	list, err := SearchInRasp("2108")
	if err != nil {
		t.Error(err)
	}
	week := 5
	uri := list[0].Url
	doc, is, gr, err := DownloadShedule(uri, week)
	if err != nil {
		t.Error(err)
	}
	shedule, err := Parse(doc, is, gr, week)
	if err != nil {
		t.Error(err)
	}

	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}

	engine, err := database.Connect(os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASS"), os.Getenv("MYSQL_DB"))
	if err != nil {
		t.Error(err)
	}
	err = UploadShedule(engine, *shedule)
	if err != nil {
		t.Error(err)
	}
}
