package ssau_parser

import (
	"log"
	"testing"
)

func TestFindInRasp(t *testing.T) {
	list := FindInRasp("2305")
	log.Println(list)
}
