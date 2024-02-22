// Создатель расписания в HTML
package htmlschedule

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"text/template"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
	"xorm.io/xorm"
)

func CreateWeekImg(
	db *xorm.Engine,
	execute string,
	now time.Time,
	user *database.TgUser,
	shedule database.Schedule,
	week int,
	botWeek int,
	caption string,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.FileBytes,
	error,
) {
	var photoFileBytes tgbotapi.FileBytes
	table, err := api.GetWeekOrdered(db, shedule, week+botWeek)
	if err != nil {
		return photoFileBytes, err
	}

	var header string
	if shedule.IsPersonal {
		header = fmt.Sprintf("Моё расписание, %d (%d) неделя", week+29, week+52)
	} else if shedule.IsGroup {
		group, err := api.GetGroup(db, shedule.ScheduleID)
		if err != nil {
			return photoFileBytes, err
		}
		header = fmt.Sprintf("%s, %d (%d) неделя", group.GroupName, week+29, week+52)
	} else {
		staff, err := api.GetStaff(db, shedule.ScheduleID)
		if err != nil {
			return photoFileBytes, err
		}
		header = fmt.Sprintf("%s %s, %d (%d) неделя", staff.FirstName, staff.LastName, week+29, week+52)
	}

	html, err := CreateHTMLShedule(db, shedule.IsGroup, header, table)
	if err != nil {
		return photoFileBytes, err
	}

	path := GeneratePath(shedule, user.L9Id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return photoFileBytes, err
		}
	}

	input := fmt.Sprintf("./%s/week_%d.html", path, week)
	output := fmt.Sprintf("./%s/week_%d.jpg", path, week)
	f, _ := os.Create(input)
	defer f.Close()
	if _, err := f.WriteString(html); err != nil {
		return photoFileBytes, err
	}

	cmd := exec.Command(execute, []string{
		"-q",
		"--width",
		"1600",
		input,
		output,
	}...) // #nosec G204
	cmd.Stderr = log.Default().Writer()
	cmd.Stdout = log.Default().Writer()
	err = cmd.Run()
	if err != nil {
		return photoFileBytes, err
	}
	photoBytes, err := os.ReadFile(output)
	if err != nil {
		return photoFileBytes, err
	}
	photoFileBytes.Bytes = photoBytes

	if err := os.Remove(output); err != nil {
		return photoFileBytes, err
	}
	if err := os.Remove(input); err != nil {
		return photoFileBytes, err
	}

	return photoFileBytes, nil
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

// Создание HTML-вёрстки расписания из таблицы
func CreateHTMLShedule(
	db *xorm.Engine,
	isGroup bool,
	header string,
	table api.WeekTable,
) (string, error) {
	data := SheduleData{
		IsGroup: isGroup,
		Header:  header,
	}
	for i, d := range table.Dates {
		data.Week = append(data.Week, WeekHead{WeekDay: shortWeekdays[i], Day: d})
	}
	tmpl, err := template.ParseFiles("templates/week_shedule.html")
	if err != nil {
		return "", err
	}

	var lessonLine [6]string
	for t, tline := range table.Pairs {

		for i, l := range tline {
			if len(l) == 0 || l[0].Type == database.Window {
				lessonLine[i] = "<th class=\"subj\"></th>\n"

				continue
			}
			line, err := LessonHTML(db, l, isGroup)
			if err != nil {
				return "", err
			}
			lessonLine[i] = line
		}
		if len(table.Times[t]) != 2 {
			return "", fmt.Errorf("incorrect WeekTable.Times element")
		}
		data.Lines = append(data.Lines,
			Line{
				Begin:   table.Times[t][0],
				End:     table.Times[t][1],
				Lessons: lessonLine,
			})
	}

	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, data)
	if err != nil {
		return "", err
	}
	html := rendered.String()

	return html, nil
}

// Вёрстка пары в HTML
func LessonHTML(db *xorm.Engine, l []database.Lesson, isGroup bool) (string, error) {
	var lessonStr string
	lessonStr += fmt.Sprintf(lessonHead, l[0].Type, l[0].Name)
	if isGroup && l[0].TeacherId != 0 {
		staff, err := api.GetStaff(db, l[0].TeacherId)
		if err != nil {
			return "", err
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
		group, err := api.GetGroup(db, l[0].GroupId)
		if err != nil {
			return "", err
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
		var err error
		lessonStr, err = addSecondSubgroup(db, lessonStr, l)
		if err != nil {
			return "", err
		}
	}
	if len(l) > 1 && !isGroup {
		for _, gr := range l[1:] {
			group, err := api.GetGroup(db, gr.GroupId)
			if err != nil {
				return "", err
			}
			lessonStr += fmt.Sprintf("<h3>%s</h3>\n", group.GroupName)
			if gr.SubGroup != 0 {
				lessonStr += fmt.Sprintf("<h3>Подгруппа: %d</h3>\n<hr>\n", l[1].SubGroup)
			}
		}
	}

	lessonStr += "</th>\n"

	return lessonStr, nil
}

func addSecondSubgroup(db *xorm.Engine, lessonStr string, l []database.Lesson) (string, error) {
	lessonStr += "<hr>\n"
	if l[0].Name != l[1].Name {
		lessonStr += fmt.Sprintf("<div><p></p></div>\n<h2>%s</h2><hr>", l[1].Name)
	}
	if l[1].TeacherId != 0 {
		staff, err := api.GetStaff(db, l[1].TeacherId)
		if err != nil {
			return "", err
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

	return lessonStr, nil
}
