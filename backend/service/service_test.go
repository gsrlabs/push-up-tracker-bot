// backend/service/mock_pushup_service.go

package service

import (
	"context"
	"testing"
	"time"

	"trackerbot/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPushupRepository struct {
	mock.Mock
}

func (m *MockPushupRepository) Pool() *pgxpool.Pool {
	args := m.Called()
	if pool, ok := args.Get(0).(*pgxpool.Pool); ok {
		return pool
	}
	return nil
}

func (m *MockPushupRepository) EnsureUser(ctx context.Context, userID int64, username string) error {
	args := m.Called(ctx, userID, username)
	return args.Error(0)
}

func (m *MockPushupRepository) AddPushups(ctx context.Context, userID int64, date time.Time, count int) error {
	args := m.Called(ctx, userID, date, count)
	return args.Error(0)
}

func (m *MockPushupRepository) GetFullStat(ctx context.Context, userID int64, date time.Time) (*repository.FullStatData, error) {
	args := m.Called(ctx, userID, date)
	if data, ok := args.Get(0).(*repository.FullStatData); ok {
		return data, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPushupRepository) GetTodayStat(ctx context.Context, userID int64, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *MockPushupRepository) GetUsername(ctx context.Context, userID int64) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockPushupRepository) SetMaxReps(ctx context.Context, userID int64, count int) error {
	args := m.Called(ctx, userID, count)
	return args.Error(0)
}

func (m *MockPushupRepository) SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPushupRepository) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockPushupRepository) ResetDailyNorm(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPushupRepository) SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error {
	args := m.Called(ctx, userID, dailyNorm)
	return args.Error(0)
}

func (m *MockPushupRepository) GetDailyNorm(ctx context.Context, userID int64) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockPushupRepository) GetFirstWorkoutDate(ctx context.Context, userID int64) (time.Time, error) {
	args := m.Called(ctx, userID)
	if date, ok := args.Get(0).(time.Time); ok {
		return date, args.Error(1)
	}
	return time.Time{}, args.Error(1)
}

func (m *MockPushupRepository) GetFirstNormCompleter(ctx context.Context, date time.Time) (int64, error) {
	args := m.Called(ctx, date)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPushupRepository) AddMaxRepsHistory(ctx context.Context, userID int64, maxReps int) error {
	args := m.Called(ctx, userID, maxReps)
	return args.Error(0)
}


func (m *MockPushupRepository) GetMaxRepsHistory(ctx context.Context, userID int64) ([]repository.MaxRepsHistoryItem, error) {
	args := m.Called(ctx, userID)
	if history, ok := args.Get(0).([]repository.MaxRepsHistoryItem); ok {
		return history, args.Error(1)
	}
	
	return nil, args.Error(1)
}

func (m *MockPushupRepository) GetMaxRepsRecord(ctx context.Context, userID int64) (repository.MaxRepsHistoryItem, error) {
	args := m.Called(ctx, userID)
	if record, ok := args.Get(0).(repository.MaxRepsHistoryItem); ok {
		return record, args.Error(1)
	}
	return repository.MaxRepsHistoryItem{}, args.Error(1)
}

func TestService_EnsureUser(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("EnsureUser", mock.Anything, int64(1), "john").
		Return(nil).
		Once()

	service := NewPushupService(mockRepo, nil, time.UTC)

	err := service.EnsureUser(context.Background(), 1, "john")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_AddPushups(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("AddPushups", mock.Anything, int64(1), mock.Anything, 10).
		Return(nil).
		Once()

	err := mockRepo.AddPushups(context.Background(), 1, time.Now(), 10)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetTodayStat(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("GetTodayStat", mock.Anything, int64(1), mock.Anything).
		Return(55, nil).
		Once()

	total, err := mockRepo.GetTodayStat(context.Background(), 1, time.Now())

	assert.NoError(t, err)
	assert.Equal(t, 55, total)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetUsername(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("GetUsername", mock.Anything, int64(5)).
		Return("john", nil).
		Once()

	username, err := mockRepo.GetUsername(context.Background(), 5)

	assert.NoError(t, err)
	assert.Equal(t, "john", username)
	mockRepo.AssertExpectations(t)
}

func TestMock_SetMaxReps(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("SetMaxReps", mock.Anything, int64(1), 50).
		Return(nil).
		Once()

	err := mockRepo.SetMaxReps(context.Background(), 1, 50)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_SetDateCompletion(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("SetDateCompletionOfDailyNorm", mock.Anything, int64(2)).
		Return(nil).
		Once()

	err := mockRepo.SetDateCompletionOfDailyNorm(context.Background(), 2)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetUserMaxReps(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("GetUserMaxReps", mock.Anything, int64(3)).
		Return(70, nil).
		Once()

	reps, err := mockRepo.GetUserMaxReps(context.Background(), 3)

	assert.NoError(t, err)
	assert.Equal(t, 70, reps)
	mockRepo.AssertExpectations(t)
}

func TestMock_ResetDailyNorm(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("ResetDailyNorm", mock.Anything, int64(4)).
		Return(nil).
		Once()

	err := mockRepo.ResetDailyNorm(context.Background(), 4)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_SetDailyNorm(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("SetDailyNorm", mock.Anything, int64(5), 120).
		Return(nil).
		Once()

	err := mockRepo.SetDailyNorm(context.Background(), 5, 120)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetDailyNorm(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("GetDailyNorm", mock.Anything, int64(6)).
		Return(150, nil).
		Once()

	norm, err := mockRepo.GetDailyNorm(context.Background(), 6)

	assert.NoError(t, err)
	assert.Equal(t, 150, norm)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetFirstWorkoutDate(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	now := time.Now()

	mockRepo.
		On("GetFirstWorkoutDate", mock.Anything, int64(7)).
		Return(now, nil).
		Once()

	date, err := mockRepo.GetFirstWorkoutDate(context.Background(), 7)

	assert.NoError(t, err)
	assert.Equal(t, now, date)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetFirstNormCompleter(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("GetFirstNormCompleter", mock.Anything, mock.Anything).
		Return(int64(99), nil).
		Once()

	userID, err := mockRepo.GetFirstNormCompleter(context.Background(), time.Now())

	assert.NoError(t, err)
	assert.Equal(t, int64(99), userID)
	mockRepo.AssertExpectations(t)
}

func TestMock_AddMaxRepsHistory(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	mockRepo.
		On("AddMaxRepsHistory", mock.Anything, int64(1), 80).
		Return(nil).
		Once()

	err := mockRepo.AddMaxRepsHistory(context.Background(), 1, 80)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetMaxRepsHistory(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	history := []repository.MaxRepsHistoryItem{
		{	
			MaxReps: 50,
		},
	}

	mockRepo.
		On("GetMaxRepsHistory", mock.Anything, int64(1)).
		Return(history, nil).
		Once()

	result, err := mockRepo.GetMaxRepsHistory(context.Background(), 1)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 50, result[0].MaxReps)
	mockRepo.AssertExpectations(t)
}

func TestMock_GetMaxRepsRecord(t *testing.T) {
	mockRepo := new(MockPushupRepository)

	record := repository.MaxRepsHistoryItem{
		MaxReps: 100,
	}

	mockRepo.
		On("GetMaxRepsRecord", mock.Anything, int64(1)).
		Return(record, nil).
		Once()

	result, err := mockRepo.GetMaxRepsRecord(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, 100, result.MaxReps)
	mockRepo.AssertExpectations(t)
}