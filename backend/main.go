package main // Объявляем пакет main - точка входа в программу

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"trackerbot/hendler"
	"trackerbot/cache"
	"trackerbot/config"
	"trackerbot/db"
	"trackerbot/repository"
	"trackerbot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	log.Printf("INFO: starting application")

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("❌ Error loading config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("❌ Invalid config: %v", err)
	}

	loc, err := time.LoadLocation(cfg.App.Timezone)
	if err != nil {
		log.Fatalf("❌ Invalid timezone: %v", err)
	}

	botToken := cfg.GetBotToken()

	if botToken == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN is not set")
	}

	fmt.Println(botToken)


	telegramBot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}

	if _, err := telegramBot.GetMe(); err != nil {
		log.Fatalf("Ошибка подключения к боту: %v", err)
	}

	log.Println("✅ Бот успешно подключен")

	telegramBot.Debug = true

	log.Printf("Авторизован как %s", telegramBot.Self.UserName)

	// DB
	db, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Panicf("Unable to connect to database: %v\n", err)
	}
	defer db.Pool.Close()

	pushupRepo := repository.NewPushupRepository(db.Pool)

	todayCache := cache.NewTodayCache(ctx, loc)

	pushupService := service.NewPushupService(pushupRepo, todayCache, loc)

	botHandler := hendler.NewBotHandler(telegramBot, pushupService)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := telegramBot.GetUpdatesChan(u)


	var wg sync.WaitGroup

	for update := range updates {
		wg.Add(1)

		go func(update tgbotapi.Update) {
			defer wg.Done()
			botHandler.HandleUpdate(update)

		}(update)

	}

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	wg.Wait()
}
