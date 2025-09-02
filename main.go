package main // Объявляем пакет main - точка входа в программу

import (
	"context"
	"fmt"
	"log"
	"os"
	
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5" // Импортируем библиотеку для работы с Telegram API
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv" // Пакет для работы с переменными окружения

	"trackerbot/bot"
	"trackerbot/cache"
	"trackerbot/repository"
	"trackerbot/service"
)

func main() {
	// Инициализация окружения и конфигурации

	// Загрузка переменных окружения из файла .env
	// Используется пакет godotenv для удобной работы с переменными окружения
	// В случае ошибки - аварийное завершение программы
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	// 1. Получение токена Telegram бота из переменных окружения
	// Безопасный способ хранения чувствительных данных
	// При отсутствии токена - аварийное завершение
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Токен не указан. Установите переменную окружения TELEGRAM_BOT_TOKEN")
	}
	fmt.Println(botToken) // Вывод токена для отладки (в продакшене следует убрать)

	// 2. Инициализация Telegram бота
	// Создание нового экземпляра бота с использованием токена
	// NewBotAPI возвращает объект BotAPI или ошибку
	telegramBot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err) // Аварийное завершение при ошибке инициализации
	}

	// Проверяем подключение к боту
    if _, err := telegramBot.GetMe(); err != nil {
        log.Fatalf("Ошибка подключения к боту: %v", err)
    }

    log.Println("✅ Бот успешно подключен")

	// 3. Настройка режима отладки
	// В режиме отладки бот выводит подробную информацию о своих действиях
	// Полезно для разработки, но в продакшене лучше отключить
	telegramBot.Debug = true

	// Логирование успешной авторизации
	// Self.UserName содержит имя вашего бота в Telegram
	log.Printf("Авторизован как %s", telegramBot.Self.UserName)

	// 4. Получение URL базы данных из переменных окружения
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" { 
		log.Fatal("DATABASE_URL не указан. Установите переменную окружения DATABASE_URL")
	}

	// 5. Инициализация пула соединений с PostgreSQL
	// pgxpool.New создает пул соединений с заданным контекстом
	// Пул соединений улучшает производительность при частых запросах
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Panicf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close() // Гарантированное закрытие соединений при завершении

	// 6. Настройка параметров пула соединений
	// Максимальное количество соединений в пуле
	dbPool.Config().MaxConns = 10
	// Максимальное время простаивания соединения
	dbPool.Config().MaxConnIdleTime = 30 * time.Minute
	
	// 7. Инициализация слоев приложения (архитектура Clean Architecture)
	
	// Репозиторий для работы с данными отжиманий
	pushupRepo := repository.NewPushupRepository(dbPool)

	// Кеш для хранения сегодняшней статистики
	todayCache := cache.NewTodayCache()

	// Запуск фоновой горутины для ежедневного сброса кеша
	go todayCache.ResetDaily()

	// Сервисный слой с бизнес-логикой
	pushupService := service.NewPushupService(*pushupRepo, todayCache)

	// Обработчик Telegram бота
	botHandler := bot.NewBotHandler(telegramBot, pushupService)

	
	
     //go botHandler.CleanupExpiredInputs() 
     // Запускаем фоновую очистку
	// 8. Настройка получения обновлений от Telegram
	// NewUpdate(0) - получаем все обновления с момента запуска
	// Timeout - таймаут длительного опроса
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

  
	// Получение канала обновлений
	updates := telegramBot.GetUpdatesChan(u)

	go botHandler.CleanupExpiredInputs()

	reminderService := service.NewReminderService(pushupService, telegramBot)
	reminderService.StartReminderChecker()

	log.Println("Сервис напоминаний запущен")

	// WaitGroup для ожидания завершения всех обработчиков
	var wg sync.WaitGroup

	// 9. Основной цикл обработки сообщений
	// Для каждого обновления запускаем обработчик в отдельной горутине
	for update := range updates {
		wg.Add(1) // Увеличиваем счетчик WaitGroup

		// Запуск анонимной функции в горутине
		go func(update tgbotapi.Update) {
			defer wg.Done() // Уменьшаем счетчик при завершении

			// Обработка обновления
			botHandler.HandleUpdate(update)
			
		}(update)

	}

	// Ожидание завершения всех обработчиков
	wg.Wait()
}
