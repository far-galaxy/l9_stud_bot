package tg

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	wkhtml "github.com/SebastiaanKlippert/go-wkhtmltopdf"
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

	html := bot.CreateHTMLShedule(week, shedule, dates, times)

	f, _ := os.Create("sh.html")
	defer f.Close()
	f.WriteString(html)

	wkhtml.SetPath("C:\\Program Files\\wkhtmltopdf\\bin\\wkhtmltopdf.exe")
	pdfg, err := wkhtml.NewPDFGenerator()
	if err != nil {
		log.Fatal(err)
	}
	pdfg.Dpi.Set(300)
	pdfg.MarginBottom.Set(0)
	pdfg.MarginTop.Set(0)
	pdfg.MarginLeft.Set(0)
	pdfg.MarginRight.Set(0)
	pdfg.Orientation.Set(wkhtml.OrientationLandscape)
	pdfg.PageSize.Set(wkhtml.PageSizeA4)
	pdfg.AddPage(wkhtml.NewPageReader(strings.NewReader(html)))

	err = pdfg.Create()
	if err != nil {
		log.Fatal(err)
	}

	err = pdfg.WriteFile("./sh.pdf")
	if err != nil {
		log.Fatal(err)
	}

	photoBytes, err := ioutil.ReadFile("./sh.pdf")
	if err != nil {
		panic(err)
	}
	fname := fmt.Sprintf("Расписание %d неделя.pdf", week-bot.Week)
	photoFileBytes := tgbotapi.FileBytes{
		Name:  fname,
		Bytes: photoBytes,
	}

	msg := tgbotapi.NewDocument(bot.TG_user.TgId, photoFileBytes)
	_, err = bot.TG.Send(msg)
	if err != nil {
		log.Println(err)
	}
	var shId int64
	if isPersonal {
		shId = 0
	} else {
		shId = shedules[0].SheduleId
	}
	markup := SummaryKeyboard(
		"week",
		shId,
		shedules[0].IsTeacher,
		dw,
	)
	str := fmt.Sprintf("Расписание на %d неделю сообщением ниже 👇", week-bot.Week)
	bot.EditOrSend(str, markup, editMsg...)
}

const head = `<html lang="ru">
<head>
<meta charset="UTF-8">
<title>Тестовая страница с расписанием</title>
<meta name='viewport' content='width=device-width,initial-scale=1'/>
<meta name="mobile-web-app-capable" content="yes">
</head>

<style>
.note div,.rasp div{background-color:#f0f8ff;padding:10px;text-align:center;border-radius:10px}.note,th.head,th.time{font-family:monospace}.subj div #text,.subj p{display:none}html{font-size:1.3rem}body{background:#dc14bd}table{table-layout:fixed;width:100%;border-spacing:5px 5px}.note div{margin:10px 0}.head p,.subj p,hr{margin:0}.rasp div{transition:.3s}th.head{background-color:#0ff;border-radius:10px;padding:5px;font-size:1.05rem}th.subj,th.time{background-color:#f0f8ff;padding:10px;border-radius:10px}th.time{font-size:1.1rem}.subj h2,.subj p{font-size:.85rem}th.subj:not(.lab,.lect,.pract,.other){background-color:#a9a9a9}.subj div{border-radius:10px;padding:5px}.subj p{font-family:monospace;color:#f0f8ff}.subj h2,.subj h3,.subj h5{font-family:monospace;text-align:left;margin:5px}.subj h3{font-size:.65rem}.subj h5{font-size:.7rem;font-weight:400}.lect div{background-color:#7fff00}.pract div{background-color:#dc143c}.lab div{background-color:#8a2be2}.other div{background-color:#ff8c00}
</style>

<body>
`
const lessonHead = `<th class="subj %s" valign="top">
<div><p></p></div>
<h2>%s</h2><hr>`

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
	html += fmt.Sprintf("<div class=\"note\"><div id=\"week\">%d неделя</div></div>\n", week-bot.Week)
	html += "<table class=\"rasp\">\n<tr><th class=\"head\" style=\"width: 4rem\">Время</th>\n"

	for i, d := range dates {
		day := d.Format("02")
		html += fmt.Sprintf("<th class=\"head\">%s<p>%s</p></th>", weekdays[i], day)
	}
	html += "</tr>\n"
	for t, tline := range shedule {
		begin := times[t].Begin.Format("15:04")
		end := times[t].End.Format("15:04")
		html += fmt.Sprintf("<tr>\n<th class=\"time\">%s<hr>%s</th>", begin, end)
		for i, l := range tline {

			if len(l) > 0 {
				html += fmt.Sprintf(lessonHead, l[0].Type, l[0].Name)
				if l[0].TeacherId != 0 {
					var t database.Teacher
					bot.DB.ID(l[0].TeacherId).Get(&t)
					name := GenerateName(t)
					html += fmt.Sprintf("<h5 id=\"prep\">%s</h5>\n", name)
				}
				if l[0].Place != "" {
					html += fmt.Sprintf("<h3>%s</h3>\n", l[0].Place)
				}

				html += "</th>\n"
			} else {
				html += "<th class=\"subj\"></th>\n"
			}
			if i%7 == 6 {
				html += "</tr>\n"
			}
		}
	}
	html += "</table></body></html>"
	return html
}
