package service

import (
	"context"
	"fmt"
	"log"
	"time"
	"trackerbot/keyboard"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ReminderService struct {
	pushupService *PushupService
	bot           *tgbotapi.BotAPI
}

func NewReminderService(pushupService *PushupService, bot *tgbotapi.BotAPI) *ReminderService {
	return &ReminderService{
		pushupService: pushupService,
		bot:           bot,
	}
}

func (rs *ReminderService) StartReminderChecker() {
	go rs.checkReminders()
}

func (rs *ReminderService) checkReminders() {
	// Проверяем каждые 10 секунд для тестирования
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		// Получаем всех пользователей, которые не тренировались более 2 минут
		inactiveUsers, err := rs.getInactiveUsers(ctx)
		if err != nil {
			log.Printf("Ошибка получения неактивных пользователей: %v", err)
			continue
		}

		// Отправляем напоминания
		for _, userID := range inactiveUsers {
			rs.sendReminder(ctx, userID)
		}
	}
}

func (rs *ReminderService) getInactiveUsers(ctx context.Context) ([]int64, error) {
	query := `
    SELECT u.user_id
    FROM users u
    LEFT JOIN (
        SELECT user_id, COALESCE(SUM(count), 0) AS total_today
        FROM pushups
        WHERE date = CURRENT_DATE
        GROUP BY user_id
    ) p ON u.user_id = p.user_id
    WHERE u.notifications_enabled = TRUE
      AND COALESCE(p.total_today, 0) < u.daily_norm
    ORDER BY u.user_id;
    `

	rows, err := rs.pushupService.repo.Pool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

func (rs *ReminderService) sendReminder(ctx context.Context, userID int64) {
	// Проверяем, доступен ли чат
	if !rs.isChatAvailable(userID) {
		log.Printf("Пользователь %d недоступен для напоминаний (чат не найден или заблокирован)", userID)
		rs.disableNotificationsForUnavailableUser(ctx, userID)
		return
	}

	// Получаем норму и текущий прогресс
	var dailyNorm, totalToday int
	query := `
		SELECT u.daily_norm, COALESCE(SUM(p.count), 0) AS total_today
		FROM users u
		LEFT JOIN pushups p ON u.user_id = p.user_id AND p.date = CURRENT_DATE
		WHERE u.user_id = $1
		GROUP BY u.daily_norm;
	`
	err := rs.pushupService.repo.Pool().QueryRow(ctx, query, userID).Scan(&dailyNorm, &totalToday)
	if err != nil {
		log.Printf("Ошибка получения статистики пользователя %d: %v", userID, err)
		return
	}

	// Получаем дату последней тренировки
	lastWorkout, err := rs.getLastWorkoutDate(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения даты последней тренировки: %v", err)
		return
	}

	remaining := dailyNorm - totalToday
	if remaining <= 0 {
		// На всякий случай — если пользователь уже выполнил норму, не шлём напоминание
		return
	}

	// Формируем сообщение
	message := "⏰ Напоминание о тренировке!\n\n"
	if lastWorkout.IsZero() {
		message += "Ты ещё не начал тренироваться сегодня! 💥\n"
	} else {
		minutesSince := int(time.Since(lastWorkout).Minutes())
		message += fmt.Sprintf("Прошло уже %d минут с твоей последней тренировки.\n", minutesSince)
	}

	message += fmt.Sprintf("Тебе осталось выполнить %d отжиманий до дневной нормы (%d всего). 💪🚀", remaining, dailyNorm)
	message += "\n\nИспользуй кнопку \"+ за день\" чтобы добавить отжимания!"

	// Отправляем
	msg := tgbotapi.NewMessage(userID, message)

	
	// Получаем статус уведомлений через сервис
	notificationsEnabled, err := rs.pushupService.GetNotificationsStatus(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения статуса уведомлений для %d: %v", userID, err)
		// В случае ошибки используем значение по умолчанию
		notificationsEnabled = true
	}

	msg.ReplyMarkup = keyboard.MainKeyboard(notificationsEnabled)

	if _, err := rs.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки напоминания пользователю %d: %v", userID, err)
		if rs.isChatNotFoundError(err) {
			rs.disableNotificationsForUnavailableUser(ctx, userID)
		}
	}
}

// Проверяем, доступен ли чат с пользователем
func (rs *ReminderService) isChatAvailable(userID int64) bool {
	// Пытаемся отправить пустое сообщение для проверки доступности чата
	msg := tgbotapi.NewMessage(userID, "")
	_, err := rs.bot.Send(msg)
	return err == nil || !rs.isChatNotFoundError(err)
}

// Проверяем, является ли ошибка "chat not found"
func (rs *ReminderService) isChatNotFoundError(err error) bool {
	return err != nil && (err.Error() == "Bad Request: chat not found" ||
		err.Error() == "Bad Request: user not found" ||
		err.Error() == "Forbidden: bot was blocked by the user")
}

// Отключаем напоминания для недоступного пользователя
func (rs *ReminderService) disableNotificationsForUnavailableUser(ctx context.Context, userID int64) {
	err := rs.pushupService.DisableNotifications(ctx, userID)
	if err != nil {
		log.Printf("Ошибка отключения напоминаний для пользователя %d: %v", userID, err)
	} else {
		log.Printf("Напоминания отключены для недоступного пользователя %d", userID)
	}
}

func (rs *ReminderService) getLastWorkoutDate(ctx context.Context, userID int64) (time.Time, error) {
	// Используем поле date и предполагаем время тренировки
	query := `SELECT COALESCE(MAX(date), '0001-01-01') FROM pushups WHERE user_id = $1`
	var lastDate time.Time
	err := rs.pushupService.repo.Pool().QueryRow(ctx, query, userID).Scan(&lastDate)
	if err != nil {
		return time.Time{}, err
	}
	return lastDate, nil
}
