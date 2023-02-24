package ssau_parser

import (
	"encoding/json"
	"errors"
	"fmt"
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

func FindInRasp(query string) (RaspItems, error) {
	client := http.Client{}

	req, err := http.NewRequest("GET", "https://ssau.ru/rasp", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	csrf, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists {
		return nil, errors.New("missed csrf")
	}

	parm := url.Values{}
	parm.Add("text", query)
	req, err = http.NewRequest("POST", "https://ssau.ru/rasp/search", strings.NewReader(parm.Encode()))
	if err != nil {
		return nil, err
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
		return nil, err
	}

	var list RaspItems
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(body, &list); err != nil {
			return nil, err
		}

	} else {
		return nil, fmt.Errorf("responce: %s", resp.Status)
	}

	return list, nil
}

func Connect(uri string, week int) (*goquery.Document, error) {
	client := http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://ssau.ru%s&selectedWeek=%d", uri, week), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
