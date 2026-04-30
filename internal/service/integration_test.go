package service

import (
	"booking/internal/contextKeys"
	"booking/internal/models"
	"booking/internal/repository"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestIntegration(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18.3-alpine",
		postgres.WithDatabase("booking_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
				wait.ForListeningPort("5432/tcp"),
			).WithDeadline(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer testcontainers.TerminateContainer(pgContainer)

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	t.Logf("Postgres DSN: %s", dsn)

	pool, err := repository.ConnectPool(dsn)
	require.NoError(t, err)
	defer pool.Close()

	initSQL, err := os.ReadFile("../../postgres/init.sql")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, string(initSQL))
	require.NoError(t, err)
	t.Log("Database initialized")

	repo := repository.NewRepository(pool)
	svc := NewService(repo)

	adminCtx := context.WithValue(context.Background(), contextKeys.RoleKey, "admin")

	roomInput := models.Room{
		Name:        "Integration Test Room",
		Description: "Tested room",
		Capacity:    12,
	}

	createdRoom, err := svc.CreateRoom(adminCtx, roomInput)
	require.NoError(t, err)
	t.Logf("Room created: %s (ID: %s)", createdRoom.Name, createdRoom.Id)

	scheduleInput := models.Schedule{
		RoomId:     createdRoom.Id,
		DaysOfWeek: []int{1, 2, 3, 4, 5},
		StartTime:  "09:00",
		EndTime:    "18:00",
	}

	_, err = svc.CreateSchedule(adminCtx, scheduleInput)
	require.NoError(t, err)
	t.Log("Schedule created")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	tomorrow := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

	userCtx := context.WithValue(context.Background(), contextKeys.RoleKey, "user")
	userCtx = context.WithValue(userCtx, contextKeys.UserIdKey, userID)

	slots, err := svc.GetSlots(userCtx, createdRoom.Id, tomorrow)
	require.NoError(t, err)
	require.NotEmpty(t, slots, "Slots should not be empty")

	slotToBook := slots[0]
	t.Logf("Got available slot: %s - %s", slotToBook.Start.Format("15:04"), slotToBook.End.Format("15:04"))

	bookingInput := models.Booking{
		UserId: userID,
		SlotId: slotToBook.Id,
	}

	createdBooking, err := svc.CreateBooking(userCtx, bookingInput)
	require.NoError(t, err)

	assert.Equal(t, slotToBook.Id, createdBooking.SlotId)
	assert.Equal(t, userID, createdBooking.UserId)
	assert.Equal(t, "active", createdBooking.Status)

	t.Logf("Booking created successfully! Booking ID: %s", createdBooking.Id)

	cancelledBooking, err := svc.CancelBooking(userCtx, createdBooking.Id)
	require.NoError(t, err)

	assert.Equal(t, "cancelled", cancelledBooking.Status)
	assert.Equal(t, createdBooking.Id, cancelledBooking.Id)
	t.Logf("Booking successfully cancelled: ID=%s", cancelledBooking.Id)
	t.Log("Create Room - Schedule - Get Slots - Create Booking - Cancel Booking Scenario is Successfully tested ")
}
