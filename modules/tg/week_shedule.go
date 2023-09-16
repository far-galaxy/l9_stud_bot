package tg

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/icza/gox/timex"
)

var days = [6]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// Получить расписание на неделю
// При week == -1 неделя определяется автоматически
//
// Если isPersonal == false, то обязательно заполнение объекта shedule
//
// При isPersonal == true, объект shedule игнорируется
func (bot *Bot) GetWeekSummary(
	now time.Time,
	user *database.TgUser,
	shedule database.ShedulesInUser,
	week int,
	isPersonal bool,
	caption string,
	editMsg ...tgbotapi.Message,
) error {

	if err := bot.ActShedule(isPersonal, user, &shedule); err != nil {
		return err
	}

	isCompleted := false
	if week == -1 || week == 0 {
		_, now_week := now.ISOWeek()
		now_week -= bot.Week
		week = now_week
		// Проверяем, не закончились ли пары на этой неделе
		lessons, err := bot.GetLessons(shedule, now, 1)
		if err != nil {
			return err
		}
		if len(lessons) > 0 {
			_, lesson_week := lessons[0].Begin.ISOWeek()
			if lesson_week-bot.Week > now_week {
				week += 1
				isCompleted = true
				caption = "На этой неделе больше занятий нет\n" +
					"На фото расписание следующей недели"
			}
		}
	}

	var image database.File
	var cols []string
	if !isPersonal {
		image = database.File{
			TgId:       user.TgId,
			FileType:   database.Photo,
			IsPersonal: false,
			IsGroup:    shedule.IsGroup,
			SheduleId:  shedule.SheduleId,
			Week:       week,
		}
		cols = []string{"IsPersonal", "IsGroup"}
	} else {
		image = database.File{
			TgId:       user.TgId,
			FileType:   database.Photo,
			IsPersonal: true,
			Week:       week,
		}
		cols = []string{"IsPersonal"}
	}
	has, err := bot.DB.UseBool(cols...).Get(&image)
	if err != nil {
		return err
	}

	// Получаем дату обновления расписания
	var lastUpd time.Time
	if image.IsGroup {
		var group database.Group
		group.GroupId = image.SheduleId
		if _, err := bot.DB.Get(&group); err != nil {
			return err
		}
		lastUpd = group.LastUpd
	} else {
		var teacher database.Teacher
		teacher.TeacherId = image.SheduleId
		if _, err := bot.DB.Get(&teacher); err != nil {
			return err
		}
		lastUpd = teacher.LastUpd
	}

	if !has || image.LastUpd.Before(lastUpd) {
		// Если картинки нет, или она устарела
		if has {
			if _, err := bot.DB.Delete(&image); err != nil {
				return err
			}
		}
		return bot.CreateWeekImg(now, user, shedule, week, isPersonal, caption, editMsg...)
	} else {
		// Если всё есть, скидываем, что есть
		markup := tgbotapi.InlineKeyboardMarkup{}
		if caption == "" || (caption != "" && isCompleted) {
			markup = SummaryKeyboard(
				Week,
				shedule,
				isPersonal,
				week,
			)
		}

		_, err := bot.EditOrSend(user.TgId, caption, image.FileId, markup, editMsg...)
		return err
	}
}

func (bot *Bot) GetWeekLessons(shedule database.ShedulesInUser, week int) ([]database.Lesson, error) {
	condition := CreateCondition(shedule)

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
	shedule database.ShedulesInUser,
	week int,
	isPersonal bool,
	caption string,
	editMsg ...tgbotapi.Message,
) error {
	lessons, err := bot.GetWeekLessons(shedule, week)
	if err != nil {
		return err
	}
	if len(lessons) == 0 {
		// TODO: сделать костыль поизящнее и предупреждать, если неделя пустая
		// TODO: так же проработать нулевую неделю
		next, err := bot.GetWeekLessons(shedule, week+1)
		if err != nil {
			return err
		}
		if len(next) > 0 {
			lessons = next
			week += 1
		} else {
			return fmt.Errorf("no lessons: %d, week %d", shedule.SheduleId, week)
		}
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

	weekBegin := timex.WeekStart(lessons[0].Begin.Year(), week+bot.Week)
	for i := range days {
		dates = append(dates, weekBegin.Add(time.Hour*time.Duration(24*i)))
	}

	table := make([][6][]database.Lesson, height-minDay+1)
	pairs := GroupPairs(lessons)

	for _, p := range pairs {
		day := int(math.Floor(p[0].Begin.Sub(weekBegin).Hours() / 24))
		table[p[0].NumInShedule-minDay][day] = p
	}

	// Проверяем пустые строки и эпирическим путём подбираем для них время (или не подбираем вовсе)
	for y, line := range table {
		count := 0
		for _, l := range line {
			count += len(l)
		}
		if count == 0 {
			nilPair := ssau_parser.Pair{}
			if y == len(table) {
				times = append(times, nilPair)
			} else {
				times = append(times[:y+1], times[y:]...)
				times[y] = nilPair
			}
		}
	}

	var header string
	if isPersonal {
		header = fmt.Sprintf("Моё расписание, %d неделя", week)
	} else if shedule.IsGroup {
		var group database.Group
		if _, err := bot.DB.ID(shedule.SheduleId).Get(&group); err != nil {
			return err
		}
		header = fmt.Sprintf("%s, %d неделя", group.GroupName, week)
	} else {
		var teacher database.Teacher
		if _, err := bot.DB.ID(shedule.SheduleId).Get(&teacher); err != nil {
			return err
		}
		header = fmt.Sprintf("%s %s, %d неделя", teacher.FirstName, teacher.LastName, week)
	}

	html := bot.CreateHTMLShedule(shedule.IsGroup, header, table, dates, times)

	path := GeneratePath(shedule, isPersonal, user.L9Id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	input := fmt.Sprintf("./%s/week_%d.html", path, week)
	output := fmt.Sprintf("./%s/week_%d.jpg", path, week)
	f, _ := os.Create(input)
	defer f.Close()
	f.WriteString(html)

	cmd := exec.CommandContext(context.Background(), bot.WkPath, []string{
		"--width",
		"1600",
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
	// Качаем фото и сохраняем данные о нём в БД
	photo := tgbotapi.NewPhoto(user.TgId, photoFileBytes)
	photo.Caption = caption
	isCompleted := strings.Contains(caption, "На этой неделе больше занятий нет")
	if caption == "" || isCompleted {
		photo.ReplyMarkup = SummaryKeyboard(
			Week,
			shedule,
			isPersonal,
			week,
		)
	}
	resp, err := bot.TG.Send(photo)
	if err != nil {
		return err
	}
	file := database.File{
		FileId:     resp.Photo[0].FileID,
		FileType:   database.Photo,
		TgId:       user.TgId,
		IsPersonal: isPersonal,
		IsGroup:    shedule.IsGroup,
		SheduleId:  shedule.SheduleId,
		Week:       week,
		LastUpd:    now,
	}
	_, err = bot.DB.InsertOne(file)

	if err := os.Remove(output); err != nil {
		return err
	}
	if err := os.Remove(input); err != nil {
		return err
	}

	// Удаляем старое сообщение
	if len(editMsg) != 0 {
		del := tgbotapi.NewDeleteMessage(
			editMsg[0].Chat.ID,
			editMsg[0].MessageID,
		)
		if _, err := bot.TG.Request(del); err != nil {
			return err
		}
	}
	return err
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
.subj div,th.head,th.subj,th.time{border-radius:10px}.note,.subj p,th.head,th.time{font-family:monospace}.note div,.rasp div{background-color:#f0f8ff;padding:10px;text-align:center;border-radius:10px}.subj div #text,.subj p{display:none}html{font-size:1.5rem}body{background:black}table{table-layout:fixed;width:100%;border-spacing:5px 5px}.note div{margin:10px 0}.head p,.subj p,hr{margin:0}.rasp div{transition:.3s}th.head{background-color:#0ff;padding:5px;font-size:1.05rem}th.subj,th.time{background-color:#f0f8ff;padding:10px}th.time{font-size:1.1rem}.subj h2,.subj p{font-size:.85rem}th.subj:not(.lab,.lect,.pract,.other){background-color:#a9a9a9}.subj div{padding:5px}.subj p{color:#f0f8ff}.subj h2,.subj h3,.subj h5{font-family:monospace;text-align:left;margin:5px}.subj h3{font-size:.65rem}.subj h5{font-size:.7rem;font-weight:400}.lect div{background-color:#7fff00}.pract div{background-color:#dc143c}.lab div{background-color:#8a2be2}.mil div,.other div{background-color:#ff8c00}.window div{background-color:#00f}.cons div{background-color:green}.exam div{background-color:purple}.kurs div{background-color:orange}
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

func (bot *Bot) CreateHTMLShedule(
	isGroup bool,
	header string,
	shedule [][6][]database.Lesson,
	dates []time.Time,
	times []ssau_parser.Pair,
) string {
	html := head
	html += fmt.Sprintf("<div class=\"note\"><div id=\"week\">%s</div></div>\n", header)
	html += "<table class=\"rasp\">\n<tr><th class=\"head\" style=\"width: 4rem\">Время</th>\n"

	for i, d := range dates {
		day := d.Format("02")
		html += fmt.Sprintf("<th class=\"head\">%s<p>%s</p></th>", shortWeekdays[i], day)
	}
	html += "</tr>\n"

	for t, tline := range shedule {
		var begin, end string
		if times[t].Begin.IsZero() {
			begin = ("--:--")
			end = ("--:--")
		} else {
			begin = times[t].Begin.Format("15:04")
			end = times[t].End.Format("15:04")
		}
		html += fmt.Sprintf("<tr>\n<th class=\"time\">%s<hr>%s</th>", begin, end)
		for i, l := range tline {

			if len(l) > 0 && l[0].Type != "window" {
				html += fmt.Sprintf(lessonHead, l[0].Type, l[0].Name)
				if isGroup && l[0].TeacherId != 0 {
					var t database.Teacher
					bot.DB.ID(l[0].TeacherId).Get(&t)
					html += fmt.Sprintf("<h5 id=\"prep\">%s %s</h5>\n", t.FirstName, t.ShortName)
				}
				if l[0].Place != "" {
					html += fmt.Sprintf("<h3>%s</h3>\n", l[0].Place)
				}
				if !isGroup {
					var t database.Group
					bot.DB.ID(l[0].GroupId).Get(&t)
					html += fmt.Sprintf("<h3>%s</h3>\n", t.GroupName)
				}
				if l[0].SubGroup != 0 {
					html += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n", l[0].SubGroup)
				}
				if l[0].Comment != "" {
					html += fmt.Sprintf("<h3>%s</h3>\n", l[0].Comment)
				}

				if len(l) == 2 && isGroup {
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
						html += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n", l[1].SubGroup)
					}
					if l[1].Comment != "" {
						html += fmt.Sprintf("<h3>%s</h3>\n", l[1].Comment)
					}
				}
				if len(l) > 1 && !isGroup {
					for _, gr := range l[1:] {
						var t database.Group
						bot.DB.ID(gr.GroupId).Get(&t)
						html += fmt.Sprintf("<h3>%s</h3>\n", t.GroupName)
						if gr.SubGroup != 0 {
							html += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n<hr>\n", l[1].SubGroup)
						}
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

func (bot *Bot) CreateICS(
	user *database.TgUser,
	shedule database.ShedulesInUser,
	isPersonal bool,
	week int,
	query ...tgbotapi.CallbackQuery,
) error {
	if err := bot.ActShedule(isPersonal, user, &shedule); err != nil {
		return err
	}
	if !shedule.IsGroup {
		_, err := bot.SendMsg(
			user,
			"Скачивание .ics для преподавателей пока недоступно (:",
			GeneralKeyboard(false),
		)
		return err
	}
	lessons, err := bot.GetWeekLessons(shedule, week)
	if err != nil {
		return err
	}
	txt := "BEGIN:VCALENDAR\n" + "VERSION:2.0\n" + "CALSCALE:GREGORIAN\n" + "METHOD:REQUEST\n"
	if len(lessons) != 0 {
		for _, lesson := range lessons {
			// TODO: создать тип типов занятий
			if lesson.Type == "window" {
				continue
			}
			if lesson.Type == "mil" && !shedule.Military {
				continue
			}
			l := "BEGIN:VEVENT\n"
			l += lesson.Begin.Format("DTSTART;TZID=Europe/Samara:20060102T150405Z\n")
			l += lesson.End.Format("DTEND;TZID=Europe/Samara:20060102T150405Z\n")
			if lesson.SubGroup == 0 {
				l += fmt.Sprintf("SUMMARY:%s%s\n", Icons[lesson.Type], lesson.Name)
			} else {
				l += fmt.Sprintf(
					"SUMMARY:%s%s (%d)\n",
					Icons[lesson.Type],
					lesson.Name,
					lesson.SubGroup,
				)
			}
			var desc string
			if lesson.TeacherId != 0 {
				var t database.Teacher
				_, err := bot.DB.ID(lesson.TeacherId).Get(&t)
				if err != nil {
					return err
				}
				desc = fmt.Sprintf("%s %s\\n", t.FirstName, t.LastName)
			}
			if lesson.Comment != "" {
				desc += fmt.Sprintf("%s\\n", lesson.Comment)
			}
			l += fmt.Sprintf("DESCRIPTION:%s\n", desc)
			if lesson.Type != "mil" {
				l += fmt.Sprintf("LOCATION:%s / %s\n", Comm[lesson.Type], lesson.Place)
			}
			l += "END:VEVENT\n"
			txt += l
		}
		txt += "END:VCALENDAR"

		var fileName string
		if isPersonal {
			fileName = fmt.Sprintf("personal_%d.ics", week)
		} else {
			fileName = fmt.Sprintf("group_%d_%d.ics", shedule.SheduleId, week)
		}

		icsFileBytes := tgbotapi.FileBytes{
			Name:  fileName,
			Bytes: []byte(txt),
		}

		doc := tgbotapi.NewDocument(user.TgId, icsFileBytes)
		doc.Caption = "📖 Инструкция: https://bit.ly/ics_upload\n\n" +
			"‼️ Удалите старые занятия данной недели из календаря, если они есть"
		_, err := bot.TG.Send(doc)
		if err != nil {
			return err
		}
		if len(query) != 0 {
			ans := tgbotapi.NewCallback(query[0].ID, "")
			if _, err := bot.TG.Request(ans); err != nil {
				return err
			}
		}
	}
	return nil
}
