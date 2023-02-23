package ssau_parser

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type RaspList struct {
	Items []RaspItems
}

type RaspItems []struct {
	Id   int64
	Url  string
	Text string
}

func FindInRasp(query string) RaspItems {
	client := http.Client{}

	req, err := http.NewRequest("GET", "https://ssau.ru/rasp", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	csrf, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists {
		log.Fatal("Missed CSRF")
	}

	parm := url.Values{}
	parm.Add("text", query)
	req, err = http.NewRequest("POST", "https://ssau.ru/rasp/search", strings.NewReader(parm.Encode()))
	if err != nil {
		log.Fatal(err)
	}

	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-CSRF-TOKEN", csrf)

	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	var list RaspItems
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(body, &list); err != nil {
			log.Fatal(err)
		}

	} else {
		log.Fatal("Responce: " + resp.Status)
	}

	return list
}
