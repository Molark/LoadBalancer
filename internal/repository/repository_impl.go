package repository

import (
	"booking/internal/contextKeys"
	"booking/internal/models"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Errors for returning up for response to client
var (
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInternalError     = errors.New("internal error")
	ErrNotFound          = errors.New("resource not found")
	ErrSlotNotFound      = errors.New("slot not found")
	ErrRoomNotFound      = errors.New("room not found")
	ErrSlotAlreadyBooked = errors.New("slot already booked")
	ErrBookingNotFound   = errors.New("booking not found")
	ErrScheduleExists    = errors.New("schedule already exists")
)

func ConnectPool(dsn string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		slog.Error("pgxpool.ParseConfig", slog.Any("error", err))
		return nil, err
	}
	config.MaxConns = 10
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.MinConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("pgxpool.NewWithConfig", slog.Any("error", err))
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		slog.Error("pool.Ping", slog.Any("error", err))
		return nil, err
	}
	slog.Info("Pool established")
	return pool, nil
}

func (r repositoryStruct) GetRooms(ctx context.Context) ([]models.Room, error) {
	rows, err := r.pool.Query(ctx, "SELECT id, name, description, capacity, createdat from rooms")
	if err != nil {
		slog.Error("Error in running query in GetRooms", slog.Any("error", err))
		return nil, ErrInternalError
	}
	defer rows.Close()
	rooms, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Room])
	if err != nil {
		slog.Error("Error in decoding in GetRooms", slog.Any("error", err))
		return nil, ErrInternalError
	}
	slog.Info("Returning from GetRooms", slog.Any("room", rooms))
	return rooms, nil
}

func (r repositoryStruct) CreateRoom(ctx context.Context, room models.Room) (models.Room, error) {
	query := `
		INSERT INTO rooms (name, description, capacity)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, capacity, createdat
	`
	row, err := r.pool.Query(ctx, query, room.Name, room.Description, room.Capacity)
	if err != nil {
		slog.Error("Error in running query in CreateRoom", slog.Any("error", err))
		return models.Room{}, ErrInternalError
	}
	defer row.Close()
	createdRoom, err := pgx.CollectOneRow(row, pgx.RowToStructByName[models.Room])
	if err != nil {
		slog.Error("Error in decoding in CreateRoom", slog.Any("error", err))
		return models.Room{}, ErrInternalError
	}
	slog.Info("Returning from CreateRoom", slog.Any("createdRoom", createdRoom))
	return createdRoom, nil
}

func (r repositoryStruct) CreateSchedule(ctx context.Context, schedule models.Schedule) (models.Schedule, error) {
	var created models.Schedule

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {

		var roomExists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)
		`, schedule.RoomId).Scan(&roomExists)
		if err != nil {
			slog.Error("Error checking room exists in CreateSchedule", slog.Any("error", err))
			return ErrInternalError
		}
		if !roomExists {
			return ErrRoomNotFound
		}

		var scheduleExists bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM schedules WHERE roomid = $1)
		`, schedule.RoomId).Scan(&scheduleExists)

		if err != nil {
			slog.Error("Error checking schedule exists in CreateSchedule", slog.Any("error", err))
			return ErrInternalError
		}
		if scheduleExists {
			return ErrScheduleExists
		}

		var scheduleID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO schedules (roomid, starttime, endtime)
			VALUES ($1, $2, $3)
			RETURNING id
		`, schedule.RoomId, schedule.StartTime, schedule.EndTime).Scan(&scheduleID)
		if err != nil {
			slog.Error("Error inserting schedule in CreateSchedule", slog.Any("error", err))
			return ErrInternalError
		}

		if len(schedule.DaysOfWeek) > 0 {
			for _, day := range schedule.DaysOfWeek {
				_, err = tx.Exec(ctx, `
					INSERT INTO schedule_weekdays (scheduleid, weekdayid)
					VALUES ($1, $2)
				`, scheduleID, day)
				if err != nil {
					slog.Error("Error inserting schedule_weekday in CreateSchedule",
						slog.Any("error", err),
						slog.Int("day", day),
					)
					return ErrInternalError
				}
			}
		}

		created = models.Schedule{
			Id:         scheduleID,
			RoomId:     schedule.RoomId,
			StartTime:  schedule.StartTime,
			EndTime:    schedule.EndTime,
			DaysOfWeek: schedule.DaysOfWeek,
		}

		return nil
	})

	if err != nil {
		if !errors.Is(err, ErrRoomNotFound) && !errors.Is(err, ErrScheduleExists) {
			slog.Error("CreateSchedule transaction failed", slog.Any("error", err))
			return models.Schedule{}, ErrInternalError
		}
		return models.Schedule{}, err
	}

	slog.Info("Returning from CreateSchedule", slog.Any("created", created))
	return created, nil
}

func (r repositoryStruct) GetSlots(ctx context.Context, roomId uuid.UUID, date time.Time) ([]models.Slot, error) {
	var roomExists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)
	`, roomId).Scan(&roomExists)

	if err != nil {
		slog.Error("GetSlots: failed to check room existence")
		return nil, ErrInternalError
	}

	if !roomExists {
		return nil, ErrRoomNotFound
	}
	err = r.EnsureSlotsForDay(ctx, roomId, date)
	if err != nil {
		return nil, err
	}
	startOfDay := date.UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := r.pool.Query(ctx, `
		SELECT id, roomid, starttime, endtime
		FROM slots
		WHERE roomid = $1
			AND starttime >= $2
		  	AND starttime < $3
		  	AND booked = FALSE
		ORDER BY starttime
	`, roomId, startOfDay, endOfDay)

	if err != nil {
		slog.Error("GetSlots: query failed", slog.Any("error", err))
		return nil, ErrInternalError
	}
	defer rows.Close()

	slots, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Slot])
	if err != nil {
		slog.Error("GetSlots: failed to scan rows", slog.Any("error", err))
		return nil, ErrInternalError
	}

	slog.Info("GetSlots: success")

	return slots, nil
}

func (r repositoryStruct) CreateBooking(ctx context.Context, booking models.Booking) (models.Booking, error) {

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var slotExists bool
		var isBooked bool
		var startTime time.Time

		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM slots WHERE id = $1)
		`, booking.SlotId).Scan(&slotExists)

		if err != nil {
			slog.Error("CreateBooking: failed to check existence", slog.Any("error", err))
			return ErrInternalError
		}

		if !slotExists {
			return ErrSlotNotFound
		}
		err = tx.QueryRow(ctx, `
 				SELECT booked, starttime FROM slots WHERE id = $1
		`, booking.SlotId).Scan(&isBooked, &startTime)
		if err != nil {
			slog.Error("CreateBooking: failed to check slot status", slog.Any("error", err))
		}
		if isBooked {
			return ErrSlotAlreadyBooked
		}
		if time.Now().After(startTime) {
			slog.Error("CreateBooking: booking is already started", slog.Any("error", err))
			return ErrInvalidRequest
		}
		var bookingID uuid.UUID
		var createdAt time.Time

		err = tx.QueryRow(ctx, `
			INSERT INTO bookings (slotid, userid, status)
			VALUES ($1, $2, 'active')
			RETURNING id, createdat
		`, booking.SlotId, booking.UserId).Scan(&bookingID, &createdAt)
		if err != nil {
			slog.Error("CreateBooking: failed to insert booking", slog.Any("error", err))
			return ErrInternalError
		}

		_, err = tx.Exec(ctx, `
			UPDATE slots 
			SET booked = TRUE 
			WHERE id = $1
		`, booking.SlotId)

		if err != nil {
			slog.Error("CreateBooking: failed to update slot booked status", slog.Any("error", err))
			return ErrInternalError
		}
		booking.Id = bookingID
		booking.CreatedAt = createdAt
		booking.Status = "active"

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrSlotNotFound) || errors.Is(err, ErrSlotAlreadyBooked) {
			return models.Booking{}, err
		}
		slog.Error("CreateBooking: transaction failed", slog.Any("error", err))
		return models.Booking{}, ErrInternalError
	}

	slog.Info("Booking created successfully")
	slog.Info("CreateBooking: success", slog.Any("booking", booking))
	return booking, nil
}

func (r repositoryStruct) GetBookings(ctx context.Context, page, pageSize int) ([]models.Booking, int, error) {

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&total)
	if err != nil {
		slog.Error("GetBookings: failed to count bookings", slog.Any("error", err))
		return nil, 0, ErrInternalError
	}

	if total == 0 {
		return []models.Booking{}, 0, nil
	}

	offset := (page - 1) * pageSize

	rows, err := r.pool.Query(ctx, `
		SELECT id, slotid, userid, status, createdat
		FROM bookings
		ORDER BY createdat DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)

	if err != nil {
		slog.Error("GetBookings: query failed", slog.Any("error", err))
		return nil, 0, ErrInternalError
	}
	defer rows.Close()

	bookings, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Booking])
	if err != nil {
		slog.Error("GetBookings: failed to scan rows", slog.Any("error", err))
		return nil, 0, ErrInternalError
	}

	slog.Info("GetBookings: success")

	return bookings, total, nil
}

func (r repositoryStruct) GetMyBookings(ctx context.Context) ([]models.Booking, error) {
	UserId := ctx.Value(contextKeys.UserIdKey).(uuid.UUID)
	rows, err := r.pool.Query(ctx, `
		SELECT b.id, b.slotid, b.userid, b.status, b.createdat
		FROM bookings b
		JOIN slots s ON b.slotid = s.id
		WHERE b.userid = $1
			AND s.starttime >= NOW() AND b.status = 'active'
	`, UserId)

	if err != nil {
		slog.Error("GetMyBookings: query failed", slog.Any("error", err))
		return nil, ErrInternalError
	}

	defer rows.Close()

	bookings, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Booking])
	if err != nil {
		slog.Error("GetMyBookings: failed to scan rows", slog.Any("error", err))
		return nil, ErrInternalError
	}

	slog.Info("GetMyBookings: success")
	return bookings, nil
}

func (r repositoryStruct) CancelBooking(ctx context.Context, bookingId uuid.UUID) (models.Booking, error) {
	var returnedBooking models.Booking
	UserId := ctx.Value(contextKeys.UserIdKey).(uuid.UUID)
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {

		var id uuid.UUID
		var slotID uuid.UUID
		var bookingUserID uuid.UUID
		var status string
		var createdAt time.Time

		err := tx.QueryRow(ctx, `
			SELECT id, slotid, userid, status, createdat
			FROM bookings
			WHERE id = $1
		`, bookingId).Scan(&id, &slotID, &bookingUserID, &status, &createdAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrBookingNotFound
			}
			slog.Error("CancelBooking: failed to query booking", slog.Any("error", err))
			return ErrInternalError
		}

		if bookingUserID != UserId {
			return ErrForbidden
		}
		if status == "cancelled" {
			returnedBooking = models.Booking{
				Id:        id,
				SlotId:    slotID,
				UserId:    bookingUserID,
				Status:    status,
				CreatedAt: createdAt,
			}
			return nil
		}

		_, err = tx.Exec(ctx, `
			UPDATE bookings 
			SET status = 'cancelled' 
			WHERE id = $1
		`, bookingId)

		if err != nil {
			slog.Error("CancelBooking: failed to update booking status", slog.Any("error", err))
			return ErrInternalError
		}

		_, err = tx.Exec(ctx, `
			UPDATE slots 
			SET booked = FALSE 
			WHERE id = $1
		`, slotID)

		if err != nil {
			slog.Error("CancelBooking: failed to reset slot booked = FALSE", slog.Any("error", err))
			return ErrInternalError
		}

		returnedBooking = models.Booking{
			Id:        id,
			SlotId:    slotID,
			UserId:    bookingUserID,
			Status:    "cancelled",
			CreatedAt: createdAt,
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrBookingNotFound) || errors.Is(err, ErrForbidden) {
			return models.Booking{}, err
		}
		slog.Error("CancelBooking: transaction failed", slog.Any("error", err))
		return models.Booking{}, ErrInternalError
	}

	slog.Info("Booking cancelled successfully", slog.Any("bookingId", bookingId))
	return returnedBooking, nil
}

func (r repositoryStruct) EnsureSlotsForDay(ctx context.Context, roomId uuid.UUID, date time.Time) error {
	startOfDay := date.UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var slotsExist bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM slots 
			WHERE roomid = $1 
				AND starttime >= $2 
				AND starttime < $3
		)
	`, roomId, startOfDay, endOfDay).Scan(&slotsExist)
	if err != nil {
		slog.Error("EnsureSlotsForDay: failed to check slots existence", slog.Any("error", err))
		return ErrInternalError
	}
	if slotsExist {
		return nil
	}

	var scheduleExists bool
	err = r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM schedules WHERE roomid = $1)
	`, roomId).Scan(&scheduleExists)

	if err != nil {
		slog.Error("EnsureSlotsForDay: failed to check schedule existence", slog.Any("error", err))
		return ErrInternalError
	}
	if !scheduleExists {
		return nil
	}

	var schID uuid.UUID
	var start, end time.Time
	err = r.pool.QueryRow(ctx, `
		SELECT id, starttime, endtime 
		FROM schedules 
		WHERE roomid = $1
	`, roomId).Scan(&schID, &start, &end)
	if err != nil {
		slog.Error("EnsureSlotsForDay: failed to fetch schedule", slog.Any("error", err))
		return ErrInternalError
	}

	wd := int(date.UTC().Weekday())
	if wd == 0 {
		wd = 7
	}

	var isWorkingDay bool
	err = r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM schedule_weekdays 
			WHERE scheduleid = $1 AND weekdayid = $2
		)
	`, schID, wd).Scan(&isWorkingDay)

	if err != nil {
		slog.Error("EnsureSlotsForDay: failed to check working day", slog.Any("error", err))
		return ErrInternalError
	}
	if !isWorkingDay {
		return nil
	}

	startTime := startOfDay.Add(
		time.Duration(start.Hour())*time.Hour +
			time.Duration(start.Minute())*time.Minute +
			time.Duration(start.Second())*time.Second,
	)
	endTime := startOfDay.Add(
		time.Duration(end.Hour())*time.Hour +
			time.Duration(end.Minute())*time.Minute +
			time.Duration(end.Second())*time.Second,
	)

	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		current := startTime
		for {
			slotEnd := current.Add(30 * time.Minute)
			if slotEnd.After(endTime) {
				break
			}

			_, errExec := tx.Exec(ctx, `
				INSERT INTO slots (roomid, starttime, endtime)
				VALUES ($1, $2, $3)
			`, roomId, current, slotEnd)

			if errExec != nil {
				slog.Error("EnsureSlotsForDay: failed to insert slot")
				return ErrInternalError
			}

			current = slotEnd
		}
		return nil
	})

	if err != nil {
		slog.Error("EnsureSlotsForDay: transaction failed", slog.Any("error", err))
		return ErrInternalError
	}

	slog.Info("EnsureSlotsForDay: slots created successfully")
	return nil
}
func (r repositoryStruct) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	query := `
		INSERT INTO users (email, passwordHash, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, passwordhash, role, createdat
	`
	row, err := r.pool.Query(ctx, query, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		slog.Error("CreateUser: query failed", slog.Any("error", err))
		return models.User{}, ErrInternalError
	}
	defer row.Close()

	created, err := pgx.CollectOneRow(row, pgx.RowToStructByName[models.User])
	if err != nil {
		slog.Error("CreateUser: scan failed", slog.Any("error", err))
		return models.User{}, ErrInternalError
	}
	return created, nil
}

func (r repositoryStruct) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	query := `
		SELECT id, email, passwordHash, role, createdat 
		FROM users 
		WHERE email = $1
	`
	row, err := r.pool.Query(ctx, query, email)
	if err != nil {
		slog.Error("GetUserByEmail: query failed", slog.Any("error", err))
		return models.User{}, ErrInternalError
	}
	defer row.Close()

	user, err := pgx.CollectOneRow(row, pgx.RowToStructByName[models.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		slog.Error("GetUserByEmail: scan failed", slog.Any("error", err))
		return models.User{}, ErrInternalError
	}
	return user, nil
}
