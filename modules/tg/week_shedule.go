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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/icza/gox/timex"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
)

var days = [6]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// Получить расписание на неделю
//
// При week == -1 неделя определяется автоматически
func (bot *Bot) GetWeekSummary(
	now time.Time,
	shedule database.Schedule,
	week int,
	caption string,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {

	if _, err := bot.ActShedule(&shedule); err != nil {
		return nilMsg, err
	}

	isCompleted, err := bot.CheckWeek(now, &week, shedule)
	if err != nil {
		return nilMsg, err
	}
	if isCompleted {
		caption = "На этой неделе больше занятий нет\n" +
			"На фото расписание следующей недели"
	}

	var image database.File
	var cols []string
	if !shedule.IsPersonal {
		image = database.File{
			TgId:       shedule.TgUser.TgId,
			FileType:   database.Photo,
			IsPersonal: false,
			IsGroup:    shedule.IsGroup,
			SheduleId:  shedule.ScheduleID,
			Week:       week,
		}
		cols = []string{"IsPersonal", "IsGroup"}
	} else {
		image = database.File{
			TgId:       shedule.TgUser.TgId,
			FileType:   database.Photo,
			IsPersonal: true,
			Week:       week,
		}
		cols = []string{"IsPersonal"}
	}
	has, err := bot.DB.UseBool(cols...).Get(&image)
	if err != nil {
		return nilMsg, err
	}

	// Получаем дату обновления расписания
	var lastUpd time.Time
	if image.IsGroup {
		group, err := api.GetGroup(bot.DB, image.SheduleId)
		if err != nil {
			return nilMsg, err
		}
		lastUpd = group.LastUpd
	} else {
		staff, err := api.GetStaff(bot.DB, image.SheduleId)
		if err != nil {
			return nilMsg, err
		}
		lastUpd = staff.LastUpd
	}

	if !has || image.LastUpd.Before(lastUpd) {
		// Если картинки нет, или она устарела
		if has {
			if _, err := bot.DB.Delete(&image); err != nil {
				return nilMsg, err
			}
		}

		err := bot.CreateWeekImg(now, shedule.TgUser, shedule, week, caption, editMsg...)
		if err != nil {
			markup := SummaryKeyboard(
				Week,
				shedule,
				week,
				false,
			)

			return bot.SendMsg(shedule.TgUser, "Возникла ошибка при создании изображения", markup)
		}

		return nilMsg, err
	}
	// Если всё есть, скидываем, что есть
	markup := tgbotapi.InlineKeyboardMarkup{}
	if (caption == "" || (caption != "" && isCompleted)) && shedule.TgUser.TgId > 0 {
		connectButton := !shedule.IsPersonal && !bot.IsThereUserShedule(shedule.TgUser)
		markup = SummaryKeyboard(
			Week,
			shedule,
			week,
			connectButton,
		)
	}

	return bot.EditOrSend(shedule.TgUser.TgId, caption, image.FileId, markup, editMsg...)
}

// Проверка, не закончились ли пары на этой неделе
//
// При week == -1 неделя определяется автоматически
func (bot *Bot) CheckWeek(now time.Time, week *int, schedule database.Schedule) (bool, error) {
	if *week == -1 || *week == 0 {
		_, nowWeek := now.ISOWeek()
		nowWeek -= bot.Week
		*week = nowWeek
		lesson, err := api.GetNearLesson(bot.DB, schedule, now)
		if err != nil {
			return false, err
		}
		if lesson.LessonId != 0 {
			_, lessonWeek := lesson.Begin.ISOWeek()
			if lessonWeek-bot.Week > nowWeek {
				*week++

				return true, nil
			}

			return false, nil
		}
	}

	return false, nil
}

func (bot *Bot) CreateWeekImg(
	now time.Time,
	user *database.TgUser,
	shedule database.Schedule,
	week int,
	caption string,
	editMsg ...tgbotapi.Message,
) error {
	lessons, err := api.GetWeekLessons(bot.DB, shedule, week+bot.Week)
	if err != nil {
		return err
	}
	if len(lessons) == 0 {
		next, err := api.GetWeekLessons(bot.DB, shedule, week+bot.Week+1)
		if err != nil {
			return err
		}
		if len(next) > 0 {
			lessons = next
			week++
		} else {
			return fmt.Errorf("no lessons: %d, week %d", shedule.ScheduleID, week)
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
	if shedule.IsPersonal {
		header = fmt.Sprintf("Моё расписание, %d неделя", week)
	} else if shedule.IsGroup {
		group, err := api.GetGroup(bot.DB, shedule.ScheduleID)
		if err != nil {
			return err
		}
		header = fmt.Sprintf("%s, %d неделя", group.GroupName, week)
	} else {
		staff, err := api.GetStaff(bot.DB, shedule.ScheduleID)
		if err != nil {
			return err
		}
		header = fmt.Sprintf("%s %s, %d неделя", staff.FirstName, staff.LastName, week)
	}

	html, err := bot.CreateHTMLShedule(shedule.IsGroup, header, table, dates, times)
	if err != nil {
		return err
	}

	path := GeneratePath(shedule, user.L9Id)
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
		"-q",
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
	connectButton := !shedule.IsPersonal && !bot.IsThereUserShedule(user)
	if (caption == "" || isCompleted) && user.TgId > 0 {
		photo.ReplyMarkup = SummaryKeyboard(
			Week,
			shedule,
			week,
			connectButton,
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
		IsPersonal: shedule.IsPersonal,
		IsGroup:    shedule.IsGroup,
		SheduleId:  shedule.ScheduleID,
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

func GeneratePath(sh database.Schedule, userID int64) string {
	var path string
	if sh.IsPersonal {
		path = fmt.Sprintf("personal/%d", userID)
	} else if sh.IsGroup {
		path = fmt.Sprintf("group/%d", sh.ScheduleID)
	} else {
		path = fmt.Sprintf("staff/%d", sh.ScheduleID)
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

const lessonHead = `<th class="subj %s" valign="top">
<div><p></p></div>
<h2>%s</h2><hr>`

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
	Begin   time.Time
	End     time.Time
	Lessons [6]string
}

// TODO: подумать о своём поведении и сделать эти процессы покрасивее
func (bot *Bot) CreateHTMLShedule(
	isGroup bool,
	header string,
	shedule [][6][]database.Lesson,
	dates []time.Time,
	times []ssauparser.Pair,
) (string, error) {
	data := SheduleData{
		IsGroup: isGroup,
		Header:  header,
	}
	for i, d := range dates {
		data.Week = append(data.Week, WeekHead{WeekDay: shortWeekdays[i], Day: d})
	}
	tmpl, err := template.ParseFiles("templates/week_shedule.html")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return "", err
		}
		bot.Debug.Println(err)
	}

	var lessonLine [6]string
	for t, tline := range shedule {

		for i, l := range tline {
			if len(l) == 0 || l[0].Type == database.Window {
				lessonLine[i] = "<th class=\"subj\"></th>\n"

				continue
			}

			lessonLine[i] = LessonHTML(bot, l, isGroup)
		}
		data.Lines = append(data.Lines,
			Line{
				Begin:   times[t].Begin,
				End:     times[t].End,
				Lessons: lessonLine,
			})
	}

	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, data)
	if err != nil {
		bot.Debug.Println(err)
	}
	html := rendered.String()

	return html, nil
}

// Вёрстка пары в HTML
func LessonHTML(bot *Bot, l []database.Lesson, isGroup bool) string {
	var lessonStr string
	lessonStr += fmt.Sprintf(lessonHead, l[0].Type, l[0].Name)
	if isGroup && l[0].TeacherId != 0 {
		staff, err := api.GetStaff(bot.DB, l[0].TeacherId)
		if err != nil {
			bot.Debug.Println(err)
		}
		lessonStr += fmt.Sprintf(
			"<h5 id=\"prep\">%s %s</h5>\n",
			staff.FirstName, staff.ShortName,
		)
	}
	if l[0].Place != "" {
		lessonStr += fmt.Sprintf("<h3>%s</h3>\n", l[0].Place)
	}
	if !isGroup {
		group, err := api.GetGroup(bot.DB, l[0].GroupId)
		if err != nil {
			bot.Debug.Println(err)
		}
		lessonStr += fmt.Sprintf("<h3>%s</h3>\n", group.GroupName)
	}
	if l[0].SubGroup != 0 {
		lessonStr += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n", l[0].SubGroup)
	}
	if l[0].Comment != "" {
		lessonStr += fmt.Sprintf("<h3>%s</h3>\n", l[0].Comment)
	}

	if len(l) == 2 && isGroup {
		lessonStr = addSecondSubgroup(lessonStr, l, bot)
	}
	if len(l) > 1 && !isGroup {
		for _, gr := range l[1:] {
			group, err := api.GetGroup(bot.DB, gr.GroupId)
			if err != nil {
				bot.Debug.Println(err)
			}
			lessonStr += fmt.Sprintf("<h3>%s</h3>\n", group.GroupName)
			if gr.SubGroup != 0 {
				lessonStr += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n<hr>\n", l[1].SubGroup)
			}
		}
	}

	lessonStr += "</th>\n"

	return lessonStr
}

func addSecondSubgroup(lessonStr string, l []database.Lesson, bot *Bot) string {
	lessonStr += "<hr>\n"
	if l[0].Name != l[1].Name {
		lessonStr += fmt.Sprintf("<div><p></p></div>\n<h2>%s</h2><hr>", l[1].Name)
	}
	if l[1].TeacherId != 0 {
		staff, err := api.GetStaff(bot.DB, l[1].TeacherId)
		if err != nil {
			bot.Debug.Println(err)
		}
		lessonStr += fmt.Sprintf(
			"<h5 id=\"prep\">%s %s</h5>\n",
			staff.FirstName,
			staff.ShortName,
		)
	}
	if l[1].Place != "" {
		lessonStr += fmt.Sprintf("<h3>%s</h3>\n", l[1].Place)
	}
	if l[1].SubGroup != 0 {
		lessonStr += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n", l[1].SubGroup)
	}
	if l[1].Comment != "" {
		lessonStr += fmt.Sprintf("<h3>%s</h3>\n", l[1].Comment)
	}

	return lessonStr
}
