package tg

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type InlineResult struct {
	IsGroup     bool
	SheduleID   int64
	Name        string
	Description string
}

// Обработка Inline-запроса по поиску расписания
func (bot *Bot) HandleInlineQuery(update tgbotapi.Update) (tgbotapi.Message, error) {
	isGroupChat := update.InlineQuery.ChatType == "group"
	query := update.InlineQuery
	var inlineResults []interface{}
	if len(query.Query) < 3 {
		return bot.InlineQueryError("Запрос слишком короткий...", query)
	}

	allGroups, allTeachers, siteErr := bot.SearchInDB(query.Query)
	if siteErr != nil {
		return bot.InlineQueryError("Ошибка на стороне сайта", query)
	}

	if len(allGroups) == 0 && len(allTeachers) == 0 {
		return bot.InlineQueryError("Ничего не найдено ):", query)
	}

	// Постфикс ника бота для групповых чатов
	postfix := ""
	if isGroupChat {
		postfix = fmt.Sprintf("@%s", bot.Name)
	}

	// Записываем результаты с книжечку
	var results []InlineResult
	for _, res := range allGroups {
		results = append(results, InlineResult{
			IsGroup:     true,
			SheduleID:   res.GroupId,
			Name:        res.GroupName,
			Description: res.SpecName,
		})
	}
	for _, res := range allTeachers {
		results = append(results, InlineResult{
			IsGroup:     false,
			SheduleID:   res.TeacherId,
			Name:        fmt.Sprintf("%s %s", res.FirstName, res.LastName),
			Description: res.SpecName,
		})
	}

	if len(results) > 50 {
		return bot.InlineQueryError("Слишком много результатов... Уточните запрос", query)
	}

	// Создаём итоговый список результатов
	for i, res := range results {
		command := "/staff"
		if res.IsGroup {
			command = "/group"
		}
		q := fmt.Sprintf("%s%s %d", command, postfix, res.SheduleID)

		result := tgbotapi.NewInlineQueryResultArticleHTML(
			fmt.Sprintf("%d", i+1),
			res.Name,
			q,
		)
		result.Description = res.Description

		inlineResults = append(inlineResults, result)
	}

	return bot.SendInlineResult(query.ID, inlineResults)
}

// Вывод какой-либо ошибки в результат Inline-запроса
func (bot *Bot) InlineQueryError(text string, query *tgbotapi.InlineQuery) (tgbotapi.Message, error) {
	var inlineResults []interface{}
	inlineResults = append(inlineResults, tgbotapi.NewInlineQueryResultArticleHTML(
		"1",
		text,
		text,
	))

	return bot.SendInlineResult(query.ID, inlineResults)
}

// Отправка результатов Inline-запроса
func (bot *Bot) SendInlineResult(queryID string, results []interface{}) (tgbotapi.Message, error) {
	ans := tgbotapi.InlineConfig{
		InlineQueryID: queryID,
		IsPersonal:    true,
		CacheTime:     0,
		Results:       results,
	}
	_, err := bot.TG.Request(ans)

	return nilMsg, err
}
