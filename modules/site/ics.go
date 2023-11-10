package site

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func GetICS(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/ics/") // Удаление первого слэша

	filePath := fmt.Sprintf("./shedules/ics/%s", path)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Файл не найден", http.StatusNotFound)

		return
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Ошибка чтения файла", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", path),
	)

	// Отправка содержимого файла в ответе
	if _, err := w.Write(fileContent); err != nil {
		http.Error(w, "", http.StatusInternalServerError)

		return
	}
}
