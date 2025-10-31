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
	// Проверяем каждые 48 часов
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		// Получаем всех пользователей, которые не тренировались более 2 дней
		inactiveUsers, err := rs.pushupService.GetUsersForReminder(ctx)
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

func (rs *ReminderService) sendReminder(ctx context.Context, userID int64) {
	// Проверяем, доступен ли чат
	if !rs.isChatAvailable(userID) {
		log.Printf("Пользователь %d недоступен для напоминаний (чат не найден или заблокирован)", userID)
		rs.disableNotificationsForUnavailableUser(ctx, userID)
		return
	}

	// Единый запрос со всеми данными
    var dailyNorm, totalToday int
    var lastDailyNormDate, lastWorkoutDate time.Time
    query := `
        SELECT 
            u.daily_norm,
            u.last_updated,
            COALESCE((
                SELECT SUM(count) 
                FROM pushups 
                WHERE user_id = u.user_id AND date = CURRENT_DATE
            ), 0) as total_today,
            COALESCE((
                SELECT MAX(date) 
                FROM pushups 
                WHERE user_id = u.user_id
            ), '0001-01-01'::DATE) as last_workout_date
        FROM users u
        WHERE u.user_id = $1`

    err := rs.pushupService.repo.Pool().QueryRow(ctx, query, userID).Scan(
        &dailyNorm, 
        &lastDailyNormDate, 
        &totalToday, 
        &lastWorkoutDate,
    )
    if err != nil {
        log.Printf("Ошибка получения статистики пользователя %d: %v", userID, err)
        return
    }

    remaining := dailyNorm - totalToday
    if remaining <= 0 {
        return
    }

    now := time.Now().UTC()
    hoursWithoutDailyNorm := int(now.Sub(lastDailyNormDate).Hours())
    hoursWithoutWorkout := int(now.Sub(lastWorkoutDate).Hours())

    message := rs.buildReminderMessage(remaining, dailyNorm, hoursWithoutDailyNorm, hoursWithoutWorkout, lastWorkoutDate)
    
    msg := tgbotapi.NewMessage(userID, message)
    notificationsEnabled, _ := rs.pushupService.GetNotificationsStatus(ctx, userID)
    msg.ReplyMarkup = keyboard.MainKeyboard(notificationsEnabled)

    if _, err := rs.bot.Send(msg); err != nil {
        log.Printf("Ошибка отправки напоминания пользователю %d: %v", userID, err)
        if rs.isChatNotFoundError(err) {
            rs.disableNotificationsForUnavailableUser(ctx, userID)
        }
    } else {
        rs.pushupService.UpdateLastNotification(ctx, userID)
        log.Printf("Напоминание отправлено пользователю %d", userID)
    }
}

func (rs *ReminderService) buildReminderMessage(remaining, dailyNorm, hoursWithoutDailyNorm, hoursWithoutWorkout int, lastWorkoutDate time.Time) string {
    message := "⏰ Напоминание о тренировке!\n\n"
    
    today := time.Now().UTC().Truncate(24 * time.Hour)
    trainedToday := !lastWorkoutDate.Before(today)
    
    if !trainedToday {
        message += "Ты ещё не начал тренироваться сегодня! 💥\n"
    } else {
        // Разбиваем на дни и часы
        daysWithoutNorm := hoursWithoutDailyNorm / 24
        hoursWithoutNorm := hoursWithoutDailyNorm % 24
        
        daysWithoutWorkout := hoursWithoutWorkout / 24
        hoursRemainingWorkout := hoursWithoutWorkout % 24
        
        // Форматируем с помощью наших функций
        var normPeriod, workoutPeriod string
        
        if daysWithoutNorm > 0 && hoursWithoutNorm > 0 {
            normPeriod = fmt.Sprintf("%s и %s", 
                FormatDaysCompact(daysWithoutNorm), 
                FormatHoursCompact(hoursWithoutNorm))
        } else if daysWithoutNorm > 0 {
            normPeriod = FormatDaysCompact(daysWithoutNorm)
        } else if hoursWithoutNorm > 0 {
            normPeriod = FormatHoursCompact(hoursWithoutNorm)
        } else {
            normPeriod = "менее часа"
        }
        
        if daysWithoutWorkout > 0 && hoursRemainingWorkout > 0 {
            workoutPeriod = fmt.Sprintf("%s и %s", 
                FormatDaysCompact(daysWithoutWorkout), 
                FormatHoursCompact(hoursRemainingWorkout))
        } else if daysWithoutWorkout > 0 {
            workoutPeriod = FormatDaysCompact(daysWithoutWorkout)
        } else if hoursRemainingWorkout > 0 {
            workoutPeriod = FormatHoursCompact(hoursRemainingWorkout)
        } else {
            workoutPeriod = "менее часа"
        }
        
        message += fmt.Sprintf("Прошло %s с момента выполнения дневной нормы.\n", normPeriod)
        message += fmt.Sprintf("И %s с твоей последней тренировки.\n", workoutPeriod)
    }

    message += fmt.Sprintf("Тебе осталось выполнить %d отжиманий до дневной нормы (%d всего). 💪🚀", 
        remaining, dailyNorm)
    message += "\n\nИспользуй кнопку \"➕ Добавить отжимания\""
    
    return message
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

