package notify

import (
	"fmt"
	"log"
	"time"

	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
	"stud.l9labs.ru/bot/modules/tg"
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
	if du < 24 {
		return
	}
	log.Printf("check group %s, lastCheck %v", group.GroupName, group.LastCheck)
	group.LastCheck = now
	if _, err := bot.DB.ID(group.GroupId).Update(group); err != nil {
		log.Println(err)
	}
	sh := ssauparser.WeekShedule{
		IsGroup:   true,
		SheduleID: group.GroupId,
	}
	_, _, err := bot.LoadShedule(sh, now, false)
	if err != nil {
		log.Println(err)
	}
	// Очищаем от лишних пар
	/*
		var nAdd, nDel []database.Lesson
		_, nowWeek := now.ISOWeek()
		for _, a := range add {
			_, addWeek := a.Begin.ISOWeek()
			if a.Begin.After(now) &&
				a.GroupId == group.GroupId &&
				(addWeek == nowWeek ||
					addWeek == nowWeek+1) {
				nAdd = append(nAdd, a)
			}
		}
		for _, d := range del {
			_, delWeek := d.Begin.ISOWeek()
			if d.Begin.After(now) &&
				d.GroupId == group.GroupId &&
				(delWeek == nowWeek || delWeek == nowWeek+1) {
				nDel = append(nDel, d)
			}
		}
		if len(nAdd) > 0 || len(nDel) > 0 {
			tsh := tg.Swap(sh)
			err := tg.UpdateICS(bot, tsh)
			if err != nil {
				log.Println(err)
			}
			var str string
			str = "‼ Обнаружены изменения в расписании\n"
			str = strChanges(nAdd, str, true)
			str = strChanges(nDel, str, false)
			var users []database.TgUser
			if err := bot.DB.
				UseBool("IsGroup").
				Table("ShedulesInUser").
				Cols("tgid").
				Join("INNER", "TgUser", "TgUser.l9id = ShedulesInUser.l9id").
				Find(&users, tsh); err != nil {
				log.Println(err)
			}
			for i := range users {
				if _, err := bot.SendMsg(&users[i], str, nil); nil != err {
					log.Println(err)
				}
			}
		}
	*/
}

func strChanges(add []database.Lesson, str string, isAdd bool) string {
	addLen := len(add)
	if addLen > 0 {
		if addLen > 10 {
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
	if lesson.Type == database.Military {
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
