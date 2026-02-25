package hendler

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"trackerbot/cache"
	"trackerbot/presenter"
	"trackerbot/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


type MockBot struct {
	mock.Mock
}

type MockService struct {
	mock.Mock
}

func (m *MockBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	args := m.Called(c)
	if msg, ok := args.Get(0).(tgbotapi.Message); ok {
		return msg, args.Error(1)
	}
	return tgbotapi.Message{}, args.Error(1)
}

func (m *MockBot) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	args := m.Called(c)
	if resp, ok := args.Get(0).(*tgbotapi.APIResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockService) EnsureUser(ctx context.Context, userID int64, username string) error {
	args := m.Called(ctx, userID, username)
	return args.Error(0)
}

func (m *MockService) AddPushups(ctx context.Context, userID int64, count int) (*presenter.AddPushupsViewModel, error) {
	args := m.Called(ctx, userID, count)

	if vm, ok := args.Get(0).(*presenter.AddPushupsViewModel); ok {
		return vm, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockService) SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error {
	args := m.Called(ctx, userID, dailyNorm)
	return args.Error(0)
}

func (m *MockService) SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockService) GetDailyNorm(ctx context.Context, userID int64) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockService) UpdateMaxReps(ctx context.Context, userID int64, count int) (*presenter.MaxRepsViewModel, error) {
	args := m.Called(ctx, userID, count)

	if vm, ok := args.Get(0).(*presenter.MaxRepsViewModel); ok {
		return vm, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockService) GetMaxRepsHistory(ctx context.Context, userID int64) ([]repository.MaxRepsHistoryItem, error) {
	args := m.Called(ctx, userID)

	if history, ok := args.Get(0).([]repository.MaxRepsHistoryItem); ok {
		return history, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockService) GetMaxRepsRecord(ctx context.Context, userID int64) (repository.MaxRepsHistoryItem, error) {
	args := m.Called(ctx, userID)

	if record, ok := args.Get(0).(repository.MaxRepsHistoryItem); ok {
		return record, args.Error(1)
	}
	return repository.MaxRepsHistoryItem{}, args.Error(1)
}

func (m *MockService) ResetDailyNorm(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockService) GetFullStat(ctx context.Context, userID int64) (*presenter.FullStatViewModel, error) {
	args := m.Called(ctx, userID)

	if vm, ok := args.Get(0).(*presenter.FullStatViewModel); ok {
		return vm, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockService) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockService) GetFirstWorkoutDate(ctx context.Context, userID int64) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockService) CheckNormCompletion(ctx context.Context) (bool, string) {
	args := m.Called(ctx)
	return args.Bool(0), args.String(1)
}

func (m *MockService) BuildSchedule(
	ctx context.Context,
	userID int64,
	history []repository.MaxRepsHistoryItem,
) (bytes.Buffer, error) {

	args := m.Called(ctx, userID, history)

	if buf, ok := args.Get(0).(bytes.Buffer); ok {
		return buf, args.Error(1)
	}

	return bytes.Buffer{}, args.Error(1)
}

func (m *MockService) DebugCache() *cache.TodayCache {
	args := m.Called()

	if cacheObj, ok := args.Get(0).(*cache.TodayCache); ok {
		return cacheObj
	}
	return nil
}

func TestHandleAddPushups(t *testing.T) {
	mockService := new(MockService)
	mockBot := new(MockBot)

	handler := &BotHandler{
		bot:            mockBot,
		service:        mockService,
		inputManager:   NewInputManager(),
		numericConfigs: make(map[inputType]numericConfig),
	}

	vm := &presenter.AddPushupsViewModel{
		AddedCount: 10,
		Total:      50,
		DailyNorm:  100,
		Completed:  false,
	}

	mockService.
		On("AddPushups", mock.Anything, int64(1), 10).
		Return(vm, nil).
		Once()

	mockBot.
		On("Send", mock.AnythingOfType("tgbotapi.MessageConfig")).
		Return(tgbotapi.Message{}, nil).
		Once()

	handler.handleAddPushups(context.Background(), 1, "john", 123, 10)

	mockService.AssertExpectations(t)
	mockBot.AssertExpectations(t)
}

func TestHandleSetMaxReps(t *testing.T) {
	mockService := new(MockService)
	mockBot := new(MockBot)

	handler := &BotHandler{
		bot:            mockBot,
		service:        mockService,
		inputManager:   NewInputManager(),
		numericConfigs: make(map[inputType]numericConfig),
	}

	vm := &presenter.MaxRepsViewModel{
		Count:     50,
		DailyNorm: 150,
	}

	mockService.
		On("UpdateMaxReps", mock.Anything, int64(1), 50).
		Return(vm, nil).
		Once()

	mockBot.
		On("Send", mock.Anything).
		Return(tgbotapi.Message{}, nil).
		Once()

	handler.handleSetMaxReps(context.Background(), 1, "john", 123, 50)

	mockService.AssertExpectations(t)
	mockBot.AssertExpectations(t)
}

func TestHandleStart_NewUser(t *testing.T) {
	mockService := new(MockService)
	mockBot := new(MockBot)

	handler := NewBotHandler(mockBot, mockService)

	mockService.
		On("EnsureUser", mock.Anything, int64(1), "john").
		Return(nil)

	mockService.
		On("GetUserMaxReps", mock.Anything, int64(1)).
		Return(0, nil)

	mockBot.
		On("Send", mock.Anything).
		Return(tgbotapi.Message{MessageID: 1}, nil)

	handler.handleStart(context.Background(), 123, 1, "john", inputTypeMaxReps)

	mockService.AssertExpectations(t)
	mockBot.AssertExpectations(t)
}

func TestHandleFullStat(t *testing.T) {
	mockService := new(MockService)
	mockBot := new(MockBot)

	handler := NewBotHandler(mockBot, mockService)

	vm := &presenter.FullStatViewModel{
		TodayTotal:   20,
		TotalAllTime: 200,
	}

	mockService.
		On("GetFullStat", mock.Anything, int64(1)).
		Return(vm, nil).
		Once()

	mockBot.
		On("Send", mock.Anything).
		Return(tgbotapi.Message{}, nil).
		Once()

	handler.handleFullStat(context.Background(), 1, 123)

	mockService.AssertExpectations(t)
	mockBot.AssertExpectations(t)
}

// --- Тест handleStart ---
func TestHandleStart_ExistingUser(t *testing.T) {
	mockBot := new(MockBot)
	mockService := new(MockService)

	handler := NewBotHandler(mockBot, mockService)

	ctx := context.Background()
	chatID := int64(1234)
	userID := int64(1)
	username := "alice"

	// Мокируем сервис
	mockService.On("EnsureUser", ctx, userID, username).Return(nil)
	mockService.On("GetUserMaxReps", ctx, userID).Return(42, nil)

	// Мокируем отправку сообщения
	mockBot.On("Send", mock.Anything).Return(tgbotapi.Message{}, nil)

	handler.handleStart(ctx, chatID, userID, username, inputTypeMaxReps)

	mockService.AssertCalled(t, "EnsureUser", ctx, userID, username)
	mockService.AssertCalled(t, "GetUserMaxReps", ctx, userID)
	mockBot.AssertCalled(t, "Send", mock.Anything)
}

// --- Тест handleSetCustomNorm ---
func TestHandleSetCustomNorm(t *testing.T) {
	mockBot := new(MockBot)
	mockService := new(MockService)
	handler := NewBotHandler(mockBot, mockService)

	ctx := context.Background()
	chatID := int64(123)
	userID := int64(1)
	dailyNorm := 100

	mockService.On("SetDailyNorm", ctx, userID, dailyNorm).Return(nil)
	mockBot.On("Send", mock.Anything).Return(tgbotapi.Message{}, nil)

	handler.handleSetCustomNorm(ctx, userID, chatID, dailyNorm)

	mockService.AssertCalled(t, "SetDailyNorm", ctx, userID, dailyNorm)
	mockBot.AssertCalled(t, "Send", mock.Anything)
}

// --- Тест handleProgressHistory ---
func TestHandleProgressHistory(t *testing.T) {
	mockBot := new(MockBot)
	mockService := new(MockService)
	handler := NewBotHandler(mockBot, mockService)

	ctx := context.Background()
	chatID := int64(123)
	userID := int64(1)

	history := []repository.MaxRepsHistoryItem{
		{MaxReps: 10}, {MaxReps: 20},
	}
	fakeImage := []byte("fake image")

	mockService.On("GetMaxRepsHistory", ctx, userID).Return(history, nil)
	mockService.On("BuildSchedule", ctx, userID, history).Return(fakeImage, nil)
	mockBot.On("Send", mock.Anything).Return(tgbotapi.Message{}, nil)

	handler.handleProgressHistory(ctx, userID, chatID)

	mockService.AssertCalled(t, "GetMaxRepsHistory", ctx, userID)
	mockService.AssertCalled(t, "BuildSchedule", ctx, userID, history)
	mockBot.AssertCalled(t, "Send", mock.Anything)
}

// --- Тест handleInfo ---
func TestHandleInfo(t *testing.T) {
	mockBot := new(MockBot)
	mockService := new(MockService)
	handler := NewBotHandler(mockBot, mockService)

	chatID := int64(123)
	mockBot.On("Send", mock.Anything).Return(tgbotapi.Message{}, nil)

	handler.handleInfo(chatID)

	mockBot.AssertCalled(t, "Send", mock.Anything)
}

// --- Тест handlePendingInput ---
func TestHandlePendingInput_AllCases(t *testing.T) {
	mockBot := new(MockBot)
	mockService := new(MockService)
	handler := NewBotHandler(mockBot, mockService)

	ctx := context.Background()
	chatID := int64(100)
	userID := int64(1)
	username := "alice"

	tests := []struct {
		name      string
		inputType inputType
		text      string
		wantError bool
	}{
		// --- Некорректный ввод (буквы) ---
		{"InvalidInput_Letters", inputDayLimit, "abc", true},
		{"InvalidInput_Empty", inputDayLimit, "", true},

		// --- Меньше минимума ---
		{"BelowMin_0", inputDayLimit, "0", true},
		{"BelowMin_Negative", inputDayLimit, "-5", true},

		// --- Больше лимита ---
		{"AboveLimit_OneTimeEntry", inputDayLimit, strconv.Itoa(oneTimeEntryLimit + 1), true},
		{"AboveLimit_MaxReps", inputTypeMaxReps, strconv.Itoa(maxRepsLimit + 1), true},
		{"AboveLimit_CustomNorm", inputTypeCustomNorm, strconv.Itoa(castomDailyNormLimit + 1), true},

		// --- Корректный ввод ---
		{"Valid_OneTimeEntry", inputDayLimit, "100", false},
		{"Valid_MaxReps", inputTypeMaxReps, "50", false},
		{"Valid_CustomNorm", inputTypeCustomNorm, "200", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// --- Создаем новый InputManager и добавляем pendingInput ---
			inputManager := NewInputManager()
			inputManager.Set(chatID, PendingInput{InputType: tt.inputType, MessageID: 1, CancelMsgID: 2})
			handler.inputManager = inputManager

			// --- Настраиваем мок сервиса для корректного ввода ---
			if !tt.wantError {
				switch tt.inputType {
				case inputDayLimit:
					mockService.On("AddPushups", ctx, userID, mock.Anything).Return(
						&presenter.AddPushupsViewModel{AddedCount: 100, Total: 100, DailyNorm: 150}, nil)
				case inputTypeMaxReps:
					mockService.On("UpdateMaxReps", ctx, userID, mock.Anything).Return(
						&presenter.MaxRepsViewModel{Count: 50, Rank: "🏹 Адепт упорства"}, nil)
				case inputTypeCustomNorm:
					mockService.On("SetDailyNorm", ctx, userID, mock.Anything).Return(nil)
				}
			}

			// --- Настраиваем мок бота на любые вызовы Send ---
			mockBot.On("Send", mock.MatchedBy(func(c tgbotapi.Chattable) bool { return true })).
				Return(tgbotapi.Message{}, nil)

			// --- Вызываем handlePendingInput ---
			input, ok := handler.getPendingInput(chatID)
			assert.True(t, ok, "PendingInput должен существовать перед обработкой")
			handler.handlePendingInput(ctx, input, userID, username, chatID, tt.text)

			if tt.wantError {
				// Если ожидаем ошибку, pendingInput не удаляется
				_, exists := handler.getPendingInput(chatID)
				assert.True(t, exists, "PendingInput не должен удаляться при ошибочном вводе")
			} else {
				// Корректный ввод — pendingInput очищается
				_, exists := handler.getPendingInput(chatID)
				assert.False(t, exists, "PendingInput должен быть удален после успешного ввода")
			}

			// --- Сбрасываем мок ---
			mockService.ExpectedCalls = nil
			mockBot.ExpectedCalls = nil
		})
	}
}
