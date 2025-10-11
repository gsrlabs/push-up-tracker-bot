package service

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"trackerbot/keyboard"
)

type ProgressReminderService struct {
	pushupService *PushupService
	bot           *tgbotapi.BotAPI
}

func NewProgressReminderService(pushupService *PushupService, bot *tgbotapi.BotAPI) *ProgressReminderService {
	return &ProgressReminderService{
		pushupService: pushupService,
		bot:           bot,
	}
}

func (prs *ProgressReminderService) StartProgressChecker() {
	go prs.checkProgressReminders()
}

func (prs *ProgressReminderService) checkProgressReminders() {
	// Проверяем каждые 24 часа
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Первая проверка через 1 минуту после запуска
	time.Sleep(1 * time.Minute)

	for range ticker.C {
		ctx := context.Background()

		usersToRemind, err := prs.getUsersForProgressReminder(ctx)
		if err != nil {
			log.Printf("Ошибка получения пользователей для напоминания о прогрессе: %v", err)
			continue
		}

		log.Printf("Найдено %d пользователей для напоминания о прогрессе", len(usersToRemind))

		for _, userData := range usersToRemind {
			prs.sendProgressReminder(ctx, userData)
		}
	}
}

type UserProgressData struct {
	UserID     int64
	Username   string
	CurrentMax int
	LastUpdate time.Time
	DaysPassed int
	NextTarget int
}

func (prs *ProgressReminderService) getUsersForProgressReminder(ctx context.Context) ([]UserProgressData, error) {
	query := `
    SELECT 
        user_id,
        username,
        max_reps,
        last_updated_max_reps,
        EXTRACT(DAYS FROM (CURRENT_DATE - last_updated_max_reps)) as days_passed
    FROM users 
    WHERE notifications_enabled = TRUE
      AND max_reps > 0
      AND max_reps < 100
      AND last_updated_max_reps <= CURRENT_DATE - INTERVAL '7 days' 
    ORDER BY last_updated_max_reps ASC`

	rows, err := prs.pushupService.repo.Pool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserProgressData
	for rows.Next() {
		var user UserProgressData
		var lastUpdate time.Time

		if err := rows.Scan(&user.UserID, &user.Username, &user.CurrentMax, &lastUpdate, &user.DaysPassed); err != nil {
			return nil, err
		}

		user.LastUpdate = lastUpdate
		user.NextTarget = user.CurrentMax + CalculateNextTarget(user.CurrentMax)

		users = append(users, user)
	}

	return users, nil
}

func (prs *ProgressReminderService) sendProgressReminder(ctx context.Context, userData UserProgressData) {
	// Проверяем актуальность статуса уведомлений
	notificationsEnabled, err := prs.pushupService.GetNotificationsStatus(ctx, userData.UserID)
	if err != nil || !notificationsEnabled {
		return
	}

	// Проверяем доступность чата
	if !prs.isChatAvailable(userData.UserID) {
		log.Printf("Пользователь %d недоступен для напоминания о прогрессе", userData.UserID)
		prs.disableNotificationsForUnavailableUser(ctx, userData.UserID)
		return
	}

	message := prs.buildProgressMessage(userData)
	msg := tgbotapi.NewMessage(userData.UserID, message)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard.MainKeyboard(notificationsEnabled)

	if _, err := prs.bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки напоминания о прогрессе пользователю %d: %v", userData.UserID, err)
		if prs.isChatNotFoundError(err) {
			prs.disableNotificationsForUnavailableUser(ctx, userData.UserID)
		}
	} else {
		log.Printf("Напоминание о прогрессе отправлено пользователю %d (прошло %d дней)",
			userData.UserID, userData.DaysPassed)
	}
}

func (prs *ProgressReminderService) buildProgressMessage(userData UserProgressData) string {
	message := "📈 **Время обновить твой прогресс!**\n\n"
	message += fmt.Sprintf("Прошло уже **%d дней** с твоего последнего теста максимальных отжиманий.\n\n", userData.DaysPassed)

	message += fmt.Sprintf("💪 Твой текущий рекорд: **%d отжиманий** за подход\n", userData.CurrentMax)
	message += fmt.Sprintf("🎯 Рекомендуемая цель на эту неделю: **%d отжиманий**\n\n", userData.NextTarget)

	message += fmt.Sprintf("📊 Твой текущий ранг: **%s**\n", GetUserRank(userData.CurrentMax))

	// Показываем прогресс до следующего ранга
	repsToNextRank := GetRepsToNextRank(userData.CurrentMax)
	if repsToNextRank > 0 {
		message += fmt.Sprintf("🌟 До следующего ранга осталось: **+%d отжиманий**\n\n", repsToNextRank)
	}

	// Мотивационное сообщение в зависимости от прогресса
	if userData.DaysPassed > 14 {
		message += "🚀 Не теряй прогресс! Проверь свои силы и обнови результат!\n"
	} else if userData.DaysPassed > 30 {
		message += "💥 Давно не виделись! Пора узнать, насколько ты стал сильнее!\n"
	} else {
		message += "🔥 Пора проверить, какой прогресс ты сделал за эту неделю!\n"
	}

	message += "\nИспользуй кнопку \"🎯 Обновить прогресс\" чтобы записать новый результат!"

	return message
}

// Вспомогательные методы
func (prs *ProgressReminderService) isChatAvailable(userID int64) bool {
	msg := tgbotapi.NewMessage(userID, "")
	_, err := prs.bot.Send(msg)
	return err == nil
}

func (prs *ProgressReminderService) isChatNotFoundError(err error) bool {
	return err != nil && (err.Error() == "Bad Request: chat not found" ||
		err.Error() == "Bad Request: user not found" ||
		err.Error() == "Forbidden: bot was blocked by the user")
}

func (prs *ProgressReminderService) disableNotificationsForUnavailableUser(ctx context.Context, userID int64) {
	err := prs.pushupService.DisableNotifications(ctx, userID)
	if err != nil {
		log.Printf("Ошибка отключения напоминаний для пользователя %d: %v", userID, err)
	} else {
		log.Printf("Напоминания отключены для недоступного пользователя %d", userID)
	}
}
