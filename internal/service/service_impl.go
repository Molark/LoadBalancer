package service

import (
	"booking/internal/contextKeys"
	"booking/internal/models"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var ErrForbidden = errors.New("forbidden")
var ErrInvalidRequest = errors.New("bad request")

func (s ServiceStruct) GetRooms(ctx context.Context) ([]models.Room, error) {
	rooms, err := s.repo.GetRooms(ctx)
	if err != nil {
		slog.Error("Error in request to repository GetRooms")
		return nil, err
	}

	return rooms, nil

}
func (s ServiceStruct) CreateRoom(ctx context.Context, room models.Room) (models.Room, error) {
	role := ctx.Value(contextKeys.RoleKey)
	if role != "admin" {
		return models.Room{}, ErrForbidden
	}
	room, err := s.repo.CreateRoom(ctx, room)
	if err != nil {
		slog.Error("Error in request to repository CreateRoom")
		return models.Room{}, err
	}
	return room, nil
}
func (s ServiceStruct) CreateSchedule(ctx context.Context, schedule models.Schedule) (models.Schedule, error) {
	role := ctx.Value(contextKeys.RoleKey)
	if role != "admin" {
		return models.Schedule{}, ErrForbidden
	}
	if len(schedule.DaysOfWeek) > 7 {
		return models.Schedule{}, ErrInvalidRequest
	}
	seen := make(map[int]bool)
	for _, day := range schedule.DaysOfWeek {
		_, ok := seen[day]
		if !ok {
			seen[day] = true
		} else {
			return models.Schedule{}, ErrInvalidRequest
		}
		if day < 1 || day > 7 {
			return models.Schedule{}, ErrInvalidRequest
		}
	}
	schedule, err := s.repo.CreateSchedule(ctx, schedule)
	if err != nil {
		slog.Error("Error in request to repository CreateSchedule")
		return models.Schedule{}, err
	}
	return schedule, nil
}
func (s ServiceStruct) GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]models.Slot, error) {
	slots, err := s.repo.GetSlots(ctx, roomId, date)
	if err != nil {
		slog.Error("Error in request to repository GetSlots")
		return nil, err
	}
	return slots, nil
}

func (s ServiceStruct) CreateBooking(ctx context.Context, booking models.Booking) (models.Booking, error) {
	role := ctx.Value(contextKeys.RoleKey)
	if role == "admin" {
		return models.Booking{}, ErrForbidden
	}

	booking, err := s.repo.CreateBooking(ctx, booking)
	if err != nil {
		slog.Error("Error in request to repository CreateBooking")
		return models.Booking{}, err
	}
	return booking, nil
}

func (s ServiceStruct) GetBookings(ctx context.Context, page, pageSize int) ([]models.Booking, int, error) {
	role := ctx.Value(contextKeys.RoleKey)
	if role != "admin" {
		return nil, 0, ErrForbidden
	}
	bookings, total, err := s.repo.GetBookings(ctx, page, pageSize)
	if err != nil {
		slog.Error("Error in request to repository GetBookings")
		return nil, 0, err
	}
	return bookings, total, nil
}
func (s ServiceStruct) GetMyBookings(ctx context.Context) ([]models.Booking, error) {
	role := ctx.Value(contextKeys.RoleKey)
	if role == "admin" {
		return nil, ErrForbidden
	}
	bookings, err := s.repo.GetMyBookings(ctx)
	if err != nil {
		slog.Error("Error in request to repository GetBookings")
		return nil, err
	}
	return bookings, nil
}
func (s ServiceStruct) CancelBooking(ctx context.Context, bookingId uuid.UUID) (models.Booking, error) {
	role := ctx.Value(contextKeys.RoleKey)
	if role == "admin" {
		return models.Booking{}, ErrForbidden
	}
	booking, err := s.repo.CancelBooking(ctx, bookingId)
	if err != nil {
		slog.Error("Error in request to repository CancelBooking")
		return models.Booking{}, err
	}
	return booking, nil
}
func (s ServiceStruct) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	user, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		slog.Error("Error in request to repository CreateUser")
		return models.User{}, err
	}
	return user, nil
}

func (s ServiceStruct) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		slog.Error("Error in request to repository GetUserByEmail")
		return models.User{}, err
	}
	return user, nil
}
