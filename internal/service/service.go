package service

import (
	"booking/internal/models"
	"booking/internal/repository"
	"context"
	"time"

	"github.com/google/uuid"
)

type ServiceStruct struct {
	repo repository.Repository
}

type Service interface {
	GetRooms(ctx context.Context) ([]models.Room, error)
	CreateRoom(ctx context.Context, room models.Room) (models.Room, error)
	CreateSchedule(ctx context.Context, schedule models.Schedule) (models.Schedule, error)
	GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]models.Slot, error)
	CreateBooking(ctx context.Context, booking models.Booking) (models.Booking, error)
	GetBookings(ctx context.Context, page, pageSize int) ([]models.Booking, int, error)
	GetMyBookings(ctx context.Context) ([]models.Booking, error)
	CancelBooking(ctx context.Context, bookingId uuid.UUID) (models.Booking, error)
	CreateUser(ctx context.Context, user models.User) (models.User, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
}

func NewService(repo repository.Repository) *ServiceStruct {
	return &ServiceStruct{repo: repo}
}
