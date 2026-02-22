package main // Объявляем пакет main - точка входа в программу

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"trackerbot/bot"
	"trackerbot/cache"
	"trackerbot/config"
	"trackerbot/db"
	"trackerbot/repository"
	"trackerbot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5" // Импортируем библиотеку для работы с Telegram API
)

const configPath = "../config/config.yml"

func main() {
	// Инициализация окружения и конфигурации
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	// Обработка сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	log.Printf("INFO: starting application")

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("❌ Error loading config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("❌ Invalid config: %v", err)
	}

	botToken := cfg.GetBotToken()

	if botToken == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN is not set")
	}
	fmt.Println(botToken) // Вывод токена для отладки (в продакшене следует убрать)

	// 2. Инициализация Telegram бота
	// Создание нового экземпляра бота с использованием токена
	// NewBotAPI возвращает объект BotAPI или ошибку
	telegramBot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
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

	// DB
	db, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Panicf("Unable to connect to database: %v\n", err)
	}
	defer db.Pool.Close()

	// Репозиторий для работы с данными отжиманий
	pushupRepo := repository.NewPushupRepository(db.Pool)

	// Кеш для хранения сегодняшней статистики
	todayCache := cache.NewTodayCache()

	// Запуск фоновой горутины для ежедневного сброса кеша
	go todayCache.ResetDaily()

	// Сервисный слой с бизнес-логикой
	pushupService := service.NewPushupService(*pushupRepo, todayCache)

	// Обработчик Telegram бота
	botHandler := bot.NewBotHandler(telegramBot, pushupService)

	// Запускаем фоновую очистку
	// 8. Настройка получения обновлений от Telegram
	// NewUpdate(0) - получаем все обновления с момента запуска
	// Timeout - таймаут длительного опроса
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Получение канала обновлений
	updates := telegramBot.GetUpdatesChan(u)


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

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	// Ожидание завершения всех обработчиков
	wg.Wait()
}
