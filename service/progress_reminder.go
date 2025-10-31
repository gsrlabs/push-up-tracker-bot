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
    
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()

    // Первая проверка через 10 секунд после запуска
    time.Sleep(10 * time.Second)
    prs.forceCheck()
    
    for range ticker.C {
        prs.forceCheck()
    }
}


func (prs *ProgressReminderService) forceCheck() {
    ctx := context.Background()
    
    usersToRemind, err := prs.getUsersForProgressReminder(ctx, 7)
    if err != nil {
        log.Printf("❌ Ошибка запроса: %v", err)
        return
    }

    if len(usersToRemind) > 0 {
        log.Printf("Найдено %d пользователей для напоминания", len(usersToRemind))
        
        for _, userData := range usersToRemind {
            log.Printf("Обработка пользователя %s (ID: %d, max_reps: %d, дней: %d)", 
                userData.Username, userData.UserID, userData.CurrentMax, userData.DaysPassed)
            prs.sendProgressReminder(ctx, userData)
        }
    } else {
        log.Printf("Пользователи для напоминания не найдены")
    }
}

func (prs *ProgressReminderService) getUsersForProgressReminder(ctx context.Context, daysInterval int) ([]UserProgressData, error) {
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
      AND last_updated_max_reps <= CURRENT_DATE - INTERVAL '1 day' * $1
    ORDER BY last_updated_max_reps ASC`

    log.Printf("Выполнение запроса с интервалом %d дней", daysInterval)
    
    rows, err := prs.pushupService.repo.Pool().Query(ctx, query, daysInterval)
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
        
        log.Printf("Найден пользователь - %s: max_reps=%d, last_update=%s", 
            user.Username, user.CurrentMax, lastUpdate.Format("2006-01-02"))
    }

    return users, nil
}

// ДОБАВЛЯЕМ ТЕСТОВЫЙ МЕТОД ДЛЯ РУЧНОГО ЗАПУСКА
func (prs *ProgressReminderService) TestReminderForUser(ctx context.Context, userID int64) {
    log.Printf("Ручной запуск напоминания для пользователя %d", userID)
    
    // Принудительно обновляем дату для тестирования
    updateQuery := `UPDATE users SET last_updated_max_reps = CURRENT_DATE - INTERVAL '8 days' WHERE user_id = $1`
    _, err := prs.pushupService.repo.Pool().Exec(ctx, updateQuery, userID)
    if err != nil {
        log.Printf("❌ Ошибка обновления даты: %v", err)
        return
    }
    
    log.Printf("Дата обновлена для пользователя %d", userID)
    
    // Ждем немного и запускаем проверку
    time.Sleep(2 * time.Second)
    prs.forceCheck()
}

// Остальные методы без изменений
type UserProgressData struct {
    UserID      int64
    Username    string
    CurrentMax  int
    LastUpdate  time.Time
    DaysPassed  int
    NextTarget  int
}

func (prs *ProgressReminderService) sendProgressReminder(ctx context.Context, userData UserProgressData) {
    notificationsEnabled, err := prs.pushupService.GetNotificationsStatus(ctx, userData.UserID)
    if err != nil || !notificationsEnabled {
        log.Printf("Пользователь %d отключил уведомления", userData.UserID)
        return
    }

		// Проверяем, доступен ли чат
	if !prs.isChatAvailable(userData.UserID) {
		log.Printf("Пользователь %d недоступен для напоминаний (чат не найден или заблокирован)", userData.UserID)
		prs.disableNotificationsForUnavailableUser(ctx, userData.UserID)
		return
	}

    message := prs.buildProgressMessage(userData)
    msg := tgbotapi.NewMessage(userData.UserID, message)
    msg.ParseMode = "Markdown"
    msg.ReplyMarkup = keyboard.MainKeyboard(notificationsEnabled)

    if _, err := prs.bot.Send(msg); err != nil {
        log.Printf("❌ Ошибка отправки: %v", err)
    } else {
        log.Printf("Напоминание отправлено пользователю %d", userData.UserID)
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
    message += "\nИспользуй кнопку \"🎯 Тест максимальных отжиманий\" чтобы записать новый результат!"

    return message
}

// Проверяем, доступен ли чат с пользователем
func (prs *ProgressReminderService) isChatAvailable(userID int64) bool {
	// Пытаемся отправить пустое сообщение для проверки доступности чата
	msg := tgbotapi.NewMessage(userID, "")
	_, err := prs.bot.Send(msg)
	return err == nil || !prs.isChatNotFoundError(err)
}

// Проверяем, является ли ошибка "chat not found"
func (prs *ProgressReminderService) isChatNotFoundError(err error) bool {
	return err != nil && (err.Error() == "Bad Request: chat not found" ||
		err.Error() == "Bad Request: user not found" ||
		err.Error() == "Forbidden: bot was blocked by the user")
}

// Отключаем напоминания для недоступного пользователя
func (prs *ProgressReminderService) disableNotificationsForUnavailableUser(ctx context.Context, userID int64) {
	err := prs.pushupService.DisableNotifications(ctx, userID)
	if err != nil {
		log.Printf("Ошибка отключения напоминаний для пользователя %d: %v", userID, err)
	} else {
		log.Printf("Напоминания отключены для недоступного пользователя %d", userID)
	}
}
