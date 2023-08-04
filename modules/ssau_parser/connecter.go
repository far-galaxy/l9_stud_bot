package ssau_parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Результаты поиска
type SearchResults []struct {
	Id   int64
	Url  string
	Text string
}

// Страница с расписанием и служебными хвостами
type Page struct {
	ID      int64
	IsGroup bool
	Week    int
	Doc     *goquery.Document
}

// Адрес основного сайта (прод или тестовый)
var HeadURL = "https://ssau.ru"

// Поиск расписания группы или преподавателя через ssau.ru/rasp/search
func SearchInRasp(query string) (SearchResults, error) {
	client := http.Client{}

	// Сначала заходим на сам сайт и получаем токены, чтобы нас посчитали человеком
	req, err := http.NewRequest("GET", HeadURL+"/rasp", nil)
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
		return nil, fmt.Errorf("missed csrf: %s", req.URL)
	}

	parm := url.Values{}
	parm.Add("text", query)

	// Теперь можно обращаться к подобию API
	req, err = http.NewRequest("POST", HeadURL+"/rasp/search", strings.NewReader(parm.Encode()))
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

	var list SearchResults
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(body, &list); err != nil {
			return nil, err
		}

	} else {
		return nil, fmt.Errorf("responce %s: %s", resp.Status, req.URL)
	}

	return list, nil
}

// Загрузка страницы с расписанием из ssau.ru/rasp по URI и номеру недели (в семестре)
func DownloadShedule(uri string, week int) (Page, error) {
	var page Page
	var err error

	if len(uri) < 15 {
		return page, fmt.Errorf("uri too short, maybe its wrong: %s", uri)
	}
	page.ID, err = strconv.ParseInt(uri[14:], 0, 64)
	if err != nil {
		return page, err
	}
	page.IsGroup = strings.Contains(uri, "group")
	page.Week = week

	client := http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s&selectedWeek=%d", HeadURL, uri, week), nil)
	if err != nil {
		return page, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return page, err
	}

	if resp.StatusCode != 200 {
		return page, fmt.Errorf("responce %s: %s", resp.Status, req.URL)
	}

	page.Doc, err = goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return page, err
	}

	return page, nil
}

// Загрузка страницы с расписанием из ssau.ru/rasp по ID и номеру недели (в семестре)
func DownloadSheduleById(id int64, isGroup bool, week int) (Page, error) {
	uri := GenerateUri(id, isGroup)
	return DownloadShedule(uri, week)
}

// Создать URI по ID и условию группа/преподаватель
func GenerateUri(id int64, isGroup bool) string {
	var uri string
	if isGroup {
		uri = fmt.Sprintf("/rasp?groupId=%d", id)
	} else {
		uri = fmt.Sprintf("/rasp?staffId=%d", id)
	}
	return uri
}
