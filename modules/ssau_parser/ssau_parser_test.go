package ssau_parser

import (
	"log"
	"os"
	"testing"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"github.com/joho/godotenv"
)

func TestFindInRasp(t *testing.T) {
	list, err := FindInRasp("2305")
	if err != nil {
		t.Error(err)
	}
	log.Println(list)
}

func TestConnect(t *testing.T) {
	list, err := FindInRasp("2305")
	if err != nil {
		t.Error(err)
	}
	uri := list[0].Url
	_, _, _, err = Connect(uri, 3)
	if err != nil {
		t.Error(err)
	}
}

func TestParse(t *testing.T) {
	list, err := FindInRasp("2305")
	if err != nil {
		t.Error(err)
	}
	week := 5
	uri := list[0].Url
	doc, is, gr, err := Connect(uri, week)
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
	err = uploadShedule(engine, *shedule)
	if err != nil {
		t.Error(err)
	}
}
