package service

import (
	"booking/internal/contextKeys"
	"booking/internal/models"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetRooms(ctx context.Context) ([]models.Room, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Room), args.Error(1)
}

func (m *MockRepository) CreateRoom(ctx context.Context, room models.Room) (models.Room, error) {
	args := m.Called(ctx, room)
	return args.Get(0).(models.Room), args.Error(1)
}

func (m *MockRepository) CreateSchedule(ctx context.Context, schedule models.Schedule) (models.Schedule, error) {
	args := m.Called(ctx, schedule)
	return args.Get(0).(models.Schedule), args.Error(1)
}

func (m *MockRepository) GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]models.Slot, error) {
	args := m.Called(ctx, roomId, date)
	return args.Get(0).([]models.Slot), args.Error(1)
}

func (m *MockRepository) CreateBooking(ctx context.Context, booking models.Booking) (models.Booking, error) {
	args := m.Called(ctx, booking)
	return args.Get(0).(models.Booking), args.Error(1)
}

func (m *MockRepository) GetBookings(ctx context.Context, page, pageSize int) ([]models.Booking, int, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]models.Booking), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetMyBookings(ctx context.Context) ([]models.Booking, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Booking), args.Error(1)
}

func (m *MockRepository) CancelBooking(ctx context.Context, bookingId uuid.UUID) (models.Booking, error) {
	args := m.Called(ctx, bookingId)
	return args.Get(0).(models.Booking), args.Error(1)
}
func (m *MockRepository) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(models.User), args.Error(1)
}
func withRole(role string) context.Context {
	return context.WithValue(context.Background(), contextKeys.RoleKey, role)
}

func withUserAndRole(userID uuid.UUID, role string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKeys.UserIdKey, userID)
	ctx = context.WithValue(ctx, contextKeys.RoleKey, role)
	return ctx
}

func TestService_GetRooms(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	expected := []models.Room{
		{Id: uuid.New(), Name: "Room A"},
		{Id: uuid.New(), Name: "Room B"},
	}

	mockRepo.On("GetRooms", mock.Anything).Return(expected, nil)

	rooms, err := s.GetRooms(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expected, rooms)
	mockRepo.AssertExpectations(t)
}

func TestService_CreateRoom_Success_Admin(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")

	input := models.Room{Name: "Big Conference Room", Capacity: 15}
	created := models.Room{
		Id:        uuid.New(),
		Name:      input.Name,
		Capacity:  input.Capacity,
		CreatedAt: time.Now(),
	}

	mockRepo.On("CreateRoom", mock.Anything, input).Return(created, nil)

	result, err := s.CreateRoom(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, created, result)
	mockRepo.AssertExpectations(t)
}

func TestService_CreateRoom_Forbidden_ForUser(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("user")

	_, err := s.CreateRoom(ctx, models.Room{Name: "Secret Room"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrForbidden)
	mockRepo.AssertNotCalled(t, "CreateRoom")
}

func TestService_CreateSchedule_InvalidDaysOfWeek_Duplicate(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")

	schedule := models.Schedule{
		RoomId:     uuid.New(),
		DaysOfWeek: []int{1, 2, 2, 3},
	}

	_, err := s.CreateSchedule(ctx, schedule)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRequest)
	mockRepo.AssertNotCalled(t, "CreateSchedule")
}

func TestService_CreateSchedule_InvalidDaysOfWeek_OutOfRange(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")

	schedule := models.Schedule{
		RoomId:     uuid.New(),
		DaysOfWeek: []int{1, 8},
	}

	_, err := s.CreateSchedule(ctx, schedule)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRequest)
	mockRepo.AssertNotCalled(t, "CreateSchedule")
}

func TestService_CreateSchedule_Success_Admin(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")

	schedule := models.Schedule{
		RoomId:     uuid.New(),
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}

	returnedSchedule := schedule
	returnedSchedule.Id = uuid.New()

	mockRepo.On("CreateSchedule", mock.Anything, mock.AnythingOfType("models.Schedule")).
		Return(returnedSchedule, nil)

	result, err := s.CreateSchedule(ctx, schedule)

	require.NoError(t, err)
	assert.Equal(t, returnedSchedule.Id, result.Id)
	mockRepo.AssertExpectations(t)
}

func TestService_GetBookings_Success_Admin(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")
	page, pageSize := 1, 20

	expectedBookings := []models.Booking{
		{Id: uuid.New(), Status: "active"},
		{Id: uuid.New(), Status: "active"},
	}
	total := 42

	mockRepo.On("GetBookings", mock.Anything, page, pageSize).
		Return(expectedBookings, total, nil)

	bookings, totalRes, err := s.GetBookings(ctx, page, pageSize)

	require.NoError(t, err)
	assert.Equal(t, expectedBookings, bookings)
	assert.Equal(t, total, totalRes)
	mockRepo.AssertExpectations(t)
}

func TestService_GetBookings_Forbidden_ForUser(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("user")

	_, _, err := s.GetBookings(ctx, 1, 20)

	require.ErrorIs(t, err, ErrForbidden)
	mockRepo.AssertNotCalled(t, "GetBookings")
}

func TestService_GetMyBookings_Success_User(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	userID := uuid.New()
	ctx := withUserAndRole(userID, "user")

	expected := []models.Booking{
		{Id: uuid.New(), UserId: userID, Status: "active"},
		{Id: uuid.New(), UserId: userID, Status: "active"},
	}

	mockRepo.On("GetMyBookings", mock.Anything).Return(expected, nil)

	bookings, err := s.GetMyBookings(ctx)

	require.NoError(t, err)
	assert.Equal(t, expected, bookings)
	mockRepo.AssertExpectations(t)
}

func TestService_GetMyBookings_Forbidden_ForAdmin(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")

	_, err := s.GetMyBookings(ctx)

	require.ErrorIs(t, err, ErrForbidden)
	mockRepo.AssertNotCalled(t, "GetMyBookings")
}

func TestService_CancelBooking_Success_User(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	userID := uuid.New()
	bookingID := uuid.New()

	ctx := withUserAndRole(userID, "user")

	cancelledBooking := models.Booking{
		Id:     bookingID,
		Status: "cancelled",
	}

	mockRepo.On("CancelBooking", mock.Anything, bookingID).
		Return(cancelledBooking, nil)

	result, err := s.CancelBooking(ctx, bookingID)

	require.NoError(t, err)
	assert.Equal(t, "cancelled", result.Status)
	assert.Equal(t, bookingID, result.Id)
	mockRepo.AssertExpectations(t)
}

func TestService_CancelBooking_Forbidden_ForAdmin(t *testing.T) {
	mockRepo := new(MockRepository)
	s := NewService(mockRepo)

	ctx := withRole("admin")
	bookingID := uuid.New()

	_, err := s.CancelBooking(ctx, bookingID)

	require.ErrorIs(t, err, ErrForbidden)
	mockRepo.AssertNotCalled(t, "CancelBooking")
}
