package tg

import (
	"log"
	"os"
	"testing"
)

func TestCheckEnv(t *testing.T) {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}

	// Добавляем несуществующий ключ
	env_keys = append(env_keys, "LOST_KEY")
	if err := CheckEnv(); err != nil {
		log.Println(err)
		env_keys = env_keys[:len(env_keys)-1]
	}
}
func TestInitBot(t *testing.T) {
	if err := CheckEnv(); err != nil {
		log.Fatal(err)
	}

	_, err := InitBot("test", "TESTpass1!", "testdb", os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	// Тестируем неправильный токен
	_, err = InitBot("test", "TESTpass1!", "testdb", os.Getenv("TELEGRAM_APITOKEN")+"oops")
	if err != nil {
		log.Println(err)
	}
}

func TestInitUser(t *testing.T) {

}
