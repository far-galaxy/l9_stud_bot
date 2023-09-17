package notify

import (
	"fmt"
	"log"
	"time"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/ssau_parser"
	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/tg"
)

func CheckShedules(bot *tg.Bot, now time.Time) {
	var groups []database.Group
	if err := bot.DB.Where("groupid >= 0").Find(&groups); err != nil {
		log.Println(err)
	}
	log.Println("check changes")
	for _, group := range groups {
		CheckGroup(now, group, bot)
	}
	log.Println("check end")
}

func CheckGroup(now time.Time, group database.Group, bot *tg.Bot) {
	du := now.Sub(group.LastCheck).Hours()
	if du < 1 {
		return
	}
	log.Printf("check group %s, lastCheck %v", group.GroupName, group.LastCheck)
	group.LastCheck = now
	if _, err := bot.DB.ID(group.GroupId).Update(group); err != nil {
		log.Println(err)
	}
	sh := ssau_parser.WeekShedule{
		IsGroup:   true,
		SheduleId: group.GroupId,
	}
	add, del, err := bot.LoadShedule(sh, now, true)
	if err != nil {
		log.Println(err)
	}
	// Очищаем от лишних пар
	var n_a, n_d []database.Lesson
	_, now_week := now.ISOWeek()
	for _, a := range add {
		_, a_week := a.Begin.ISOWeek()
		if a.GroupId == group.GroupId &&
			(a_week == now_week || a_week == now_week+1) {
			n_a = append(n_a, a)
		}
	}
	for _, d := range del {
		_, d_week := d.Begin.ISOWeek()
		if d.GroupId == group.GroupId &&
			(d_week == now_week || d_week == now_week+1) {
			n_d = append(n_d, d)
		}
	}
	if len(n_a) > 0 || len(n_d) > 0 {
		str := "‼ Обнаружены изменения в расписании\n"
		str = strChanges(n_a, str, true, group.GroupId)
		str = strChanges(n_d, str, false, group.GroupId)
		var users []database.TgUser
		if err := bot.DB.
			UseBool("isgroup").
			Table("ShedulesInUser").
			Cols("tgid").
			Join("INNER", "TgUser", "TgUser.l9id = ShedulesInUser.l9id").
			Find(&users, tg.Swap(sh)); err != nil {
			log.Println(err)
		}
		for _, user := range users {
			if _, err := bot.SendMsg(&user, str, tg.GeneralKeyboard(true)); nil != err {
				log.Println(err)
			}
		}
	}
}

func strChanges(add []database.Lesson, str string, isAdd bool, group int64) string {
	add_len := len(add)
	if add_len > 0 {
		if add_len > 10 {
			add = add[:10]
		}
		if isAdd {
			str += "➕ Добавлено:\n"
		} else {
			str += "➖ Удалено:\n"
		}
		for _, a := range add {
			str += ShortPairStr(a)
		}
		/*
			if add_len > 0 {
				str += fmt.Sprintf("\nВсего замен: %d\n\n", add_len)
			}
		*/
	}
	return str
}

func ShortPairStr(lesson database.Lesson) string {
	beginStr := fmt.Sprintf(lesson.Begin.Format("02 %s 15:04"), tg.Month[lesson.Begin.Month()-1])
	var endStr string
	if lesson.Type == "mil" {
		endStr = "∞"
	} else {
		endStr = lesson.End.Format("15:04")
	}
	return fmt.Sprintf(
		"📆 %s - %s\n%s%s\n-----------------\n",
		beginStr,
		endStr,
		tg.Icons[lesson.Type],
		lesson.Name,
	)
}
