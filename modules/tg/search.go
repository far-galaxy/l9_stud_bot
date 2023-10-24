package tg

import (
	"log"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/ssauparser"
	"xorm.io/builder"
	"xorm.io/xorm"
)

// Поиск расписания в БД и на сайте
func (bot *Bot) SearchInDB(query string) ([]database.Group, []database.Teacher, error) {
	var groups []database.Group
	if err := bot.DB.Where(builder.Like{"GroupName", query}).Find(&groups); err != nil {
		return nil, nil, err
	}

	var teachers []database.Teacher
	if err := bot.DB.Where(builder.Like{"FirstName", query}).Find(&teachers); err != nil {
		return nil, nil, err
	}

	// Поиск на сайте
	list, siteErr := ssauparser.SearchInRasp(query)

	// Добавляем результаты поиска на сайте к результатам из БД
	allGroups, allTeachers := AppendSearchResults(bot.DB, list, groups, teachers)

	return allGroups, allTeachers, siteErr
}

// Поиск расписания по запросу (старый способ)
func (bot *Bot) Find(now time.Time, user *database.TgUser, query string) (tgbotapi.Message, error) {

	// Проверка запроса на вшивость
	format := regexp.MustCompile(`^(\d{4}-\d{6}[D,V,Z]{1})|(\d{4})|(^[\pL-]+)$`)
	regQuery := format.FindString(query)
	if regQuery == "" {
		return bot.SendMsg(
			user,
			"❗️Неверный формат запроса\n"+
				"Нужен номер группы или фамилия преподавателя, например:\n"+
				"<b>2305</b>\n"+
				"<b>2305-240502D</b>\n"+
				"<b>Иванов</b>\n\n"+
				"Либо попробуй новый тип поиска:\n"+
				"stud.l9labs.ru/bot/about",
			nil,
		)
	}
	// Поиск в БД
	allGroups, allTeachers, siteErr := bot.SearchInDB(regQuery)

	// Если получен единственный результат, сразу выдать (подключить) расписание
	if len(allGroups) == 1 || len(allTeachers) == 1 {
		var sheduleID int64
		var isGroup bool
		if len(allGroups) == 1 {
			sheduleID = allGroups[0].GroupId
			isGroup = true
		} else {
			sheduleID = allTeachers[0].TeacherId
			isGroup = false
		}
		shedule := ssauparser.WeekShedule{
			IsGroup:   isGroup,
			SheduleID: sheduleID,
		}
		notExists, _ := ssauparser.CheckGroupOrTeacher(bot.DB, shedule)

		return bot.ReturnSummary(notExists, user.PosTag == database.Add, user, shedule, now)

		// Если получено несколько групп
	} else if len(allGroups) != 0 {
		return bot.SendMsg(
			user,
			"Вот что я нашёл\nВыбери нужную группу",
			GenerateKeyboard(GenerateGroupsArray(allGroups, user.PosTag == database.Add)),
		)
		// Если получено несколько преподавателей
	} else if len(allTeachers) != 0 {
		return bot.SendMsg(
			user,
			"Вот что я нашёл\nВыбери нужного преподавателя",
			GenerateKeyboard(GenerateTeachersArray(allTeachers, user.PosTag == database.Add)),
		)
		// Если ничего не получено
	} else {
		var txt string
		if siteErr != nil {
			bot.Debug.Printf("sasau error: %s", siteErr)
			txt = "К сожалению, у меня ничего не нашлось, а на сайте ssau.ru/rasp произошла какая-то ошибка :(\n" +
				"Повтори попытку позже"
		} else {
			txt = "К сожалению, я ничего не нашёл ):\nПроверь свой запрос"
		}

		return bot.SendMsg(
			user,
			txt,
			nilKey,
		)
	}
}

// Добавление к результатам поиска в БД результатов поиска на сайте
func AppendSearchResults(
	db *xorm.Engine,
	list ssauparser.SearchResults,
	groups []database.Group,
	teachers []database.Teacher,
) (
	[]database.Group,
	[]database.Teacher,
) {
	allGroups := groups
	allTeachers := teachers
	for _, elem := range list {
		if strings.Contains(elem.URL, "group") {
			id := elem.ID
			allGroups = appendGroup(groups, id, db, allGroups)

		}
		if strings.Contains(elem.URL, "staff") {
			id := elem.ID
			allTeachers = appendTeacher(teachers, id, db, allTeachers)
		}
	}

	return allGroups, allTeachers
}

func appendTeacher(
	teachers []database.Teacher,
	id int64,
	db *xorm.Engine,
	allTeachers []database.Teacher,
) []database.Teacher {
	exists := false
	for _, teacher := range teachers {
		if id == teacher.TeacherId {
			exists = true

			break
		}
	}
	if exists {
		return allTeachers
	}
	sh := ssauparser.WeekShedule{
		IsGroup:   false,
		SheduleID: id,
	}
	if _, err := ssauparser.CheckGroupOrTeacher(db, sh); err != nil {
		log.Println(err)
	}
	var teacher database.Teacher
	if _, err := db.ID(id).Get(&teacher); err != nil {
		log.Println(err)

		return allTeachers
	}
	allTeachers = append(allTeachers, teacher)

	return allTeachers
}

func appendGroup(
	groups []database.Group,
	id int64,
	db *xorm.Engine,
	allGroups []database.Group,
) []database.Group {
	exists := false
	for _, group := range groups {
		if id == group.GroupId {
			exists = true

			break
		}
	}
	if exists {
		return allGroups
	}

	sh := ssauparser.WeekShedule{
		IsGroup:   true,
		SheduleID: id,
	}
	if _, err := ssauparser.CheckGroupOrTeacher(db, sh); err != nil {
		log.Println(err)
	}
	var group database.Group
	if _, err := db.ID(id).Get(&group); err != nil {
		log.Println(err)

		return allGroups
	}
	allGroups = append(allGroups, group)

	return allGroups
}
