package tg

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/icza/gox/timex"
)

var days = [6]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func (bot *Bot) GetWeekSummary(
	now time.Time,
	user *database.TgUser,
	shedule database.ShedulesInUser,
	dw int,
	isPersonal bool,
) error {
	_, week := now.ISOWeek()
	week += dw - bot.Week
	var image database.File
	if !isPersonal {
		image = database.File{
			IsPersonal: false,
			IsGroup:    shedule.IsGroup,
			SheduleId:  shedule.SheduleId,
			Week:       week,
		}
	} else {
		image = database.File{
			IsPersonal: true,
			Week:       week,
		}
	}
	has, err := bot.DB.Get(&image)
	if err != nil {
		return err
	}
	var lastUpd time.Time
	if shedule.IsGroup {
		var group database.Group
		if _, err := bot.DB.ID(shedule.SheduleId).Get(&group); err != nil {
			return err
		}
		lastUpd = group.LastUpd
	} else {
		var teacher database.Teacher
		if _, err := bot.DB.ID(shedule.SheduleId).Get(&teacher); err != nil {
			return err
		}
		lastUpd = teacher.LastUpd
	}

	if !has || image.LastUpd.Before(lastUpd) {
		// TODO: удалять старые фото
		var shedules []database.ShedulesInUser
		if isPersonal {
			shedules = append(shedules, database.ShedulesInUser{L9Id: user.L9Id})
			if _, err := bot.DB.Get(&shedules); err != nil {
				return err
			}
		} else {
			shedules = append(shedules, shedule)
		}
		bot.CreateWeekImg(now, user, shedules, dw, isPersonal)
	} else {
		msg := tgbotapi.NewPhoto(user.TgId, tgbotapi.FileID(image.FileId))
		bot.TG.Send(msg)
	}

	/*
		if os.IsNotExist(fileErr) || info.ModTime().Before(lastUpd) {
			bot.CreateWeekImg(now, user, shedules, dw, isPersonal)
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
			shedules[0].IsGroup,
			dw,
		)
	*/
	return nil
}

func (bot *Bot) GetWeekLessons(shedules []database.ShedulesInUser, week int) ([]database.Lesson, error) {
	condition := CreateCondition(shedules)

	var lessons []database.Lesson
	err := bot.DB.
		Where("WEEK(`begin`, 1) = ?", week+bot.Week).
		And(condition).
		OrderBy("begin").
		Find(&lessons)

	return lessons, err
}

func (bot *Bot) CreateWeekImg(
	now time.Time,
	user *database.TgUser,
	shedules []database.ShedulesInUser,
	dw int,
	isPersonal bool,
) error {
	_, week := now.ISOWeek()
	week += dw
	lessons, err := bot.GetWeekLessons(shedules, week-bot.Week)
	if err != nil {
		return err
	}
	if len(lessons) == 0 {
		return fmt.Errorf("no lessons: %d, week %d", shedules[0].SheduleId, week)
	}

	var dates []time.Time
	begins := make(map[time.Time]bool)
	ends := make(map[time.Time]bool)
	height := 0
	minDay := lessons[0].NumInShedule
	for _, lesson := range lessons {
		t := lesson.Begin
		begin := time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, t.Location())
		begins[begin] = true

		e := lesson.End
		end := time.Date(2000, 1, 1, e.Hour(), e.Minute(), 0, 0, e.Location())
		ends[end] = true

		if lesson.NumInShedule > height {
			height = lesson.NumInShedule
		} else if lesson.NumInShedule < minDay {
			minDay = lesson.NumInShedule
		}
	}
	var times []ssau_parser.Pair
	var beginsSlice []time.Time
	var endsSlice []time.Time
	for b := range begins {
		beginsSlice = append(beginsSlice, b)
	}
	for e := range ends {
		endsSlice = append(endsSlice, e)
	}
	sort.Slice(beginsSlice, func(i, j int) bool {
		return beginsSlice[i].Before(beginsSlice[j])
	})
	sort.Slice(endsSlice, func(i, j int) bool {
		return endsSlice[i].Before(endsSlice[j])
	})
	for i, b := range beginsSlice {
		sh := ssau_parser.Pair{
			Begin: b,
			End:   endsSlice[i],
		}
		times = append(times, sh)
	}

	weekBegin := timex.WeekStart(lessons[0].Begin.Year(), week)
	for i := range days {
		dates = append(dates, weekBegin.Add(time.Hour*time.Duration(24*i)))
	}

	shedule := make([][6][]database.Lesson, height-minDay+1)
	pairs := GroupPairs(lessons)

	for _, p := range pairs {
		day := int(math.Floor(p[0].Begin.Sub(weekBegin).Hours() / 24))
		shedule[p[0].NumInShedule-minDay][day] = p
	}

	var header string
	if isPersonal {
		header = fmt.Sprintf("%d неделя", week)
	} else if shedules[0].IsGroup {
		var group database.Group
		if _, err := bot.DB.ID(shedules[0].SheduleId).Get(&group); err != nil {
			return err
		}
		header = fmt.Sprintf("%s, %d неделя", group.GroupName, week)
	} else {
		var teacher database.Teacher
		if _, err := bot.DB.ID(shedules[0].SheduleId).Get(&teacher); err != nil {
			return err
		}
		header = fmt.Sprintf("%s %s, %d неделя", teacher.FirstName, teacher.LastName, week-bot.Week)
	}

	html := bot.CreateHTMLShedule(header, shedule, dates, times)

	path := GeneratePath(shedules[0], isPersonal, user.L9Id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	input := fmt.Sprintf("./%s/week_%d.html", path, week-bot.Week)
	output := fmt.Sprintf("./%s/week_%d.png", path, week-bot.Week)
	f, _ := os.Create(input)
	defer f.Close()
	f.WriteString(html)

	cmd := exec.CommandContext(context.Background(), bot.WkPath, []string{
		"--width",
		"1280",
		input,
		output,
	}...)
	cmd.Stderr = bot.Debug.Writer()
	cmd.Stdout = bot.Debug.Writer()
	err = cmd.Run()
	if err != nil {
		return err
	}

	photoBytes, err := os.ReadFile(output)
	if err != nil {
		return err
	}
	photoFileBytes := tgbotapi.FileBytes{
		Bytes: photoBytes,
	}
	photo := tgbotapi.NewPhoto(user.TgId, photoFileBytes)
	resp, err := bot.TG.Send(photo)
	if err != nil {
		return err
	}
	file := database.File{
		FileId:     resp.Photo[0].FileID,
		TgId:       user.TgId,
		IsPersonal: isPersonal,
		IsGroup:    shedules[0].IsGroup,
		SheduleId:  shedules[0].SheduleId,
		Week:       week - bot.Week,
		LastUpd:    now,
	}
	if _, err := bot.DB.InsertOne(file); err != nil {
		return err
	}

	/*
		var shId int64
		if isPersonal {
			shId = 0
		} else {
			shId = shedules[0].SheduleId
		}
		markup := SummaryKeyboard(
			"week",
			shId,
			shedules[0].IsGroup,
			dw,
		)
		_, nowWeek := time.Now().ISOWeek()
		var str string
		if week == nowWeek {
			str = fmt.Sprintf("Расписание на текущую (%d) неделю сообщением ниже 👇", week-bot.Week)
		} else if week-nowWeek == 1 {
			str = fmt.Sprintf("Расписание на следующую (%d) неделю сообщением ниже 👇", week-bot.Week)
		} else {
			str = fmt.Sprintf("Расписание на %d неделю сообщением ниже 👇", week-bot.Week)
		}
		bot.EditOrSend(str, markup, editMsg...)
	*/
	return nil
}

func GeneratePath(sh database.ShedulesInUser, isPersonal bool, userId int64) string {
	var path string
	if isPersonal {
		path = fmt.Sprintf("personal/%d", userId)
	} else if sh.IsGroup {
		path = fmt.Sprintf("group/%d", sh.SheduleId)
	} else {
		path = fmt.Sprintf("staff/%d", sh.SheduleId)
	}
	return "shedules/" + path
}

const head = `<html lang="ru">
<head>
<meta charset="UTF-8">
<title>Тестовая страница с расписанием</title>
<meta name='viewport' content='width=device-width,initial-scale=1'/>
<meta name="mobile-web-app-capable" content="yes">
</head>

<style>
.note div,.rasp div{background-color:#f0f8ff;padding:10px;text-align:center;border-radius:10px}.note,th.head,th.time{font-family:monospace}.subj div #text,.subj p{display:none}html{font-size:1.5rem}body{background:#dc14bd}table{table-layout:fixed;width:100%;border-spacing:5px 5px}.note div{margin:10px 0}.head p,.subj p,hr{margin:0}.rasp div{transition:.3s}th.head{background-color:#0ff;border-radius:10px;padding:5px;font-size:1.05rem}th.subj,th.time{background-color:#f0f8ff;padding:10px;border-radius:10px}th.time{font-size:1.1rem}.subj h2,.subj p{font-size:.85rem}th.subj:not(.lab,.lect,.pract,.other){background-color:#a9a9a9}.subj div{border-radius:10px;padding:5px}.subj p{font-family:monospace;color:#f0f8ff}.subj h2,.subj h3,.subj h5{font-family:monospace;text-align:left;margin:5px}.subj h3{font-size:.65rem}.subj h5{font-size:.7rem;font-weight:400}.lect div{background-color:#7fff00}.pract div{background-color:#dc143c}.lab div{background-color:#8a2be2}.other div{background-color:#ff8c00}.mil div{background-color:#ff8c00}.window div{background-color:blue}
</style>

<body>
`
const lessonHead = `<th class="subj %s" valign="top">
<div><p></p></div>
<h2>%s</h2><hr>`

var shortWeekdays = [6]string{
	"пн",
	"вт",
	"ср",
	"чт",
	"пт",
	"сб",
}

func (bot *Bot) CreateHTMLShedule(header string, shedule [][6][]database.Lesson, dates []time.Time, times []ssau_parser.Pair) string {
	html := head
	html += fmt.Sprintf("<div class=\"note\"><div id=\"week\">%s</div></div>\n", header)
	html += "<table class=\"rasp\">\n<tr><th class=\"head\" style=\"width: 4rem\">Время</th>\n"

	for i, d := range dates {
		day := d.Format("02")
		html += fmt.Sprintf("<th class=\"head\">%s<p>%s</p></th>", shortWeekdays[i], day)
	}
	html += "</tr>\n"
	for t, tline := range shedule {
		begin := times[t].Begin.Format("15:04")
		end := times[t].End.Format("15:04")
		html += fmt.Sprintf("<tr>\n<th class=\"time\">%s<hr>%s</th>", begin, end)
		for i, l := range tline {

			if len(l) > 0 && l[0].Type != "window" {
				html += fmt.Sprintf(lessonHead, l[0].Type, l[0].Name)
				if l[0].TeacherId != 0 {
					var t database.Teacher
					bot.DB.ID(l[0].TeacherId).Get(&t)
					html += fmt.Sprintf("<h5 id=\"prep\">%s %s</h5>\n", t.FirstName, t.ShortName)
				}
				if l[0].Place != "" {
					html += fmt.Sprintf("<h3>%s</h3>\n", l[0].Place)
				}
				if l[0].SubGroup != 0 {
					html += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n", l[0].SubGroup)
				}
				if l[0].Comment != "" {
					html += fmt.Sprintf("<h3>%s</h3>\n", l[0].Comment)
				}

				if len(l) == 2 {
					html += "<hr>\n"
					if l[0].Name != l[1].Name {
						html += fmt.Sprintf("<div><p></p></div>\n<h2>%s</h2><hr>", l[1].Name)
					}
					if l[1].TeacherId != 0 {
						var t database.Teacher
						bot.DB.ID(l[1].TeacherId).Get(&t)
						html += fmt.Sprintf("<h5 id=\"prep\">%s %s</h5>\n", t.FirstName, t.ShortName)
					}
					if l[1].Place != "" {
						html += fmt.Sprintf("<h3>%s</h3>\n", l[1].Place)
					}
					if l[1].SubGroup != 0 {
						html += fmt.Sprintf("<h3>%d</h3>\n", l[1].SubGroup)
					}
					if l[1].Comment != "" {
						html += fmt.Sprintf("<h3>%s</h3>\n", l[1].Comment)
					}
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
