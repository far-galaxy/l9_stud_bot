package tg

import (
	"fmt"
	"math"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/icza/gox/timex"
)

func (bot *Bot) GetWeekLessons(shedules []database.ShedulesInUser, week int, isRetry ...int) ([]database.Lesson, error) {
	condition := CreateCondition(shedules)

	var lessons []database.Lesson
	err := bot.DB.
		Where("WEEK(`begin`, 1) = ?", week+bot.Week).
		And(condition).
		OrderBy("begin").
		Find(&lessons)

	if err != nil {
		return nil, err
	}
	if len(lessons) > 0 {
		return lessons, nil
	} else if len(isRetry) == 0 || isRetry[0] < 2 {
		isRetry, err = bot.LoadShedule(shedules, week+bot.Week, isRetry...)
		if err != nil {
			return nil, err
		}
		dw := isRetry[0]
		return bot.GetWeekLessons(shedules, week, dw+1)
	} else {
		return nil, nil
	}
}

var days = [6]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func (bot *Bot) GetWeekSummary(shedules []database.ShedulesInUser, dw int, isPersonal bool, editMsg ...tgbotapi.Message) {
	_, week := time.Now().ISOWeek()
	week += dw
	lessons, _ := bot.GetWeekLessons(shedules, week-bot.Week)

	var dates []time.Time
	begins := make(map[time.Time]bool)
	ends := make(map[time.Time]bool)
	height := 0
	for _, lesson := range lessons {
		t := lesson.Begin
		begin := time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, t.Location())
		begins[begin] = true

		e := lesson.End
		end := time.Date(2000, 1, 1, e.Hour(), e.Minute(), 0, 0, e.Location())
		ends[end] = true

		if lesson.NumInShedule > height {
			height = lesson.NumInShedule
		}
	}
	var times []ssau_parser.Lesson
	for b := range begins {
		l := ssau_parser.Lesson{
			Begin: b,
		}
		times = append(times, l)
	}
	i := 0
	for e := range ends {
		times[i].End = e
		i++
	}

	weekBegin := timex.WeekStart(lessons[0].Begin.Year(), week)
	for i := range days {
		dates = append(dates, weekBegin.Add(time.Hour*time.Duration(24*i)))
	}

	shedule := make([][6][]database.Lesson, height+1)
	pairs := GroupPairs(lessons)

	for _, p := range pairs {
		day := int(math.Floor(p[0].Begin.Sub(weekBegin).Hours() / 24))
		shedule[p[0].NumInShedule][day] = p
	}

	bot.CreateHTMLShedule(week, shedule, dates, times)
}

const head = `<html lang="ru">
<head>
<meta charset="UTF-8">
<title>Тестовая страница с расписанием</title>
<link rel="stylesheet" href="./rasp.css\">
<meta name='viewport' content='width=device-width,initial-scale=1'/>
<meta name="mobile-web-app-capable" content="yes">
</head>

<body>
`
const lessonHead = `<div class="subj %s">\n'
<div><p></p></div>\n'
<h2>%s</h2><hr>\n`

var weekdays = [6]string{
	"пн",
	"вт",
	"ср",
	"чт",
	"пт",
	"сб",
}

func (bot *Bot) CreateHTMLShedule(week int, shedule [][6][]database.Lesson, dates []time.Time, times []ssau_parser.Lesson) string {
	html := head
	html += fmt.Sprintf("<div class=\"note\"><div id=\"week\">%d неделя</div></div>\n", week)
	html += "<div class=\"rasp\">\n<div class=\"head\">Время</div>\n"

	for i, d := range dates {
		day := d.Format("02")
		html += fmt.Sprintf("<div class=\"head\">%s<p>%s</p></div>", weekdays[i], day)
	}
	for t, tline := range shedule {
		begin := times[t].Begin.Format("15:04")
		end := times[t].End.Format("15:04")
		html += fmt.Sprintf("<div class=\"time\">%s<hr>%s</div>", begin, end)
		for _, l := range tline {
			if len(l) > 0 {
				html += fmt.Sprintf(lessonHead, l[0].Type, l[0].Name)
				html += "</div>\n"
			} else {
				html += "<div class=\"subj\"></div>\n"
			}
		}
	}
	html += "</div></body></html>"
	return html
}
