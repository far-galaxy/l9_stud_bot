package ssau_parser

import (
	"log"
	"testing"
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
	_, err = Connect(uri, 3)
	if err != nil {
		t.Error(err)
	}
}

func TestParse(t *testing.T) {
	list, err := FindInRasp("2207")
	if err != nil {
		t.Error(err)
	}
	uri := list[0].Url
	doc, err := Connect(uri, 3)
	if err != nil {
		t.Error(err)
	}
	_, err = Parse(doc)
	if err != nil {
		t.Error(err)
	}
}
