package tg

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/template"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssauparser"
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

	isCompleted, err := bot.CheckWeek(now, &week, shedule)
	if err != nil {
		return err
	}
	if isCompleted {
		caption = "На этой неделе больше занятий нет\n" +
			"На фото расписание следующей недели"
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
	}
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
	_, err = bot.EditOrSend(user.TgId, caption, image.FileId, markup, editMsg...)

	return err
}

// Проверка, не закончились ли пары на этой неделе
//
// При week == -1 неделя определяется автоматически
func (bot *Bot) CheckWeek(now time.Time, week *int, shedule database.ShedulesInUser) (bool, error) {
	if *week == -1 || *week == 0 {
		_, nowWeek := now.ISOWeek()
		nowWeek -= bot.Week
		*week = nowWeek
		lessons, err := bot.GetLessons(shedule, now, 1)
		if err != nil {
			return false, err
		}
		if len(lessons) > 0 {
			_, lessonWeek := lessons[0].Begin.ISOWeek()
			if lessonWeek-bot.Week > nowWeek {
				*week++

				return true, nil
			}

			return false, nil
		}
	}

	return false, nil
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
		next, err := bot.GetWeekLessons(shedule, week+1)
		if err != nil {
			return err
		}
		if len(next) > 0 {
			lessons = next
			week++
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
	var times []ssauparser.Pair
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
		sh := ssauparser.Pair{
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
			nilPair := ssauparser.Pair{}
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
	if _, err := f.WriteString(html); err != nil {
		return err
	}

	cmd := exec.Command(bot.WkPath, []string{
		"--width",
		"1600",
		input,
		output,
	}...) // #nosec G204
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

func GeneratePath(sh database.ShedulesInUser, isPersonal bool, userID int64) string {
	var path string
	if isPersonal {
		path = fmt.Sprintf("personal/%d", userID)
	} else if sh.IsGroup {
		path = fmt.Sprintf("group/%d", sh.SheduleId)
	} else {
		path = fmt.Sprintf("staff/%d", sh.SheduleId)
	}

	return "shedules/" + path
}

var shortWeekdays = [6]string{
	"пн",
	"вт",
	"ср",
	"чт",
	"пт",
	"сб",
}

type SheduleData struct {
	IsGroup bool
	Header  string
	Week    []WeekHead
	Lines   []Line
}

type WeekHead struct {
	WeekDay string
	Day     time.Time
}

type Line struct {
	Begin    time.Time
	End      time.Time
	Lessons  [6][]database.Lesson
	Teachers [6][]string
	Groups   [6][]string
}

func (bot *Bot) CreateHTMLShedule(
	isGroup bool,
	header string,
	shedule [][6][]database.Lesson,
	dates []time.Time,
	times []ssauparser.Pair,
) string {
	data := SheduleData{
		IsGroup: isGroup,
		Header:  header,
	}
	for i, d := range dates {
		data.Week = append(data.Week, WeekHead{WeekDay: shortWeekdays[i], Day: d})
	}
	tmpl, err := template.ParseFiles("templates/week_shedule.html")
	if err != nil {
		bot.Debug.Println(err)
	}

	for t, tline := range shedule {
		var teachers, groups [6][]string

		for i, l := range tline {
			if len(l) == 0 || l[0].Type == database.Window {
				continue
			}

			if isGroup {
				for p := range l {
					var t database.Teacher
					if _, err := bot.DB.ID(l[p].TeacherId).Get(&t); err != nil {
						bot.Debug.Println(err)
					}
					teachers[i] = append(teachers[i], fmt.Sprintf("%s %s", t.FirstName, t.ShortName))
				}
			} else {
				for p := range l {
					var g database.Group
					if _, err := bot.DB.ID(l[0].GroupId).Get(&g); err != nil {
						bot.Debug.Println(err)
					}
					groups[p] = append(groups[p], g.GroupName)
				}
			}
		}
		data.Lines = append(data.Lines,
			Line{
				Begin:    times[t].Begin,
				End:      times[t].End,
				Lessons:  tline,
				Teachers: teachers,
			})
	}

	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, data)
	if err != nil {
		bot.Debug.Println(err)
	}
	html := rendered.String()

	return html
}

// Создание и отправка .ics файла с расписанием указанной недели для приложений календаря
//
// При week == -1 неделя определяется автоматически
func (bot *Bot) CreateICS(
	now time.Time,
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

	isCompleted, err := bot.CheckWeek(now, &week, shedule)
	if err != nil {
		return err
	}
	lessons, err := bot.GetWeekLessons(shedule, week)
	if err != nil {
		return err
	}
	txt := "BEGIN:VCALENDAR\n" + "VERSION:2.0\n" + "CALSCALE:GREGORIAN\n" + "METHOD:REQUEST\n"
	if len(lessons) != 0 {
		for _, lesson := range lessons {
			if lesson.Type == database.Window {
				continue
			}
			if lesson.Type == database.Military && !shedule.Military {
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
			if lesson.Type != database.Military {
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
		if isCompleted {
			doc.Caption = "А это файл для приложения календаря:\n https://bit.ly/ics_upload"
		} else {
			doc.Caption = "📖 Инструкция: https://bit.ly/ics_upload\n\n" +
				"‼️ Удалите старые занятия данной недели из календаря, если они есть"
		}
		doc.ReplyMarkup = bot.AutoGenKeyboard(user)
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
