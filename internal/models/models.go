package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"passwordhash" json:"-"`
	Role         string    `db:"role" json:"role"`
	CreatedAt    time.Time `db:"createdat" json:"createdAt"`
}

type Room struct {
	Id          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Capacity    int       `db:"capacity" json:"capacity"`
	CreatedAt   time.Time `db:"createdat" json:"createdAt"`
}

type Schedule struct {
	Id         uuid.UUID `db:"id" json:"id"`
	RoomId     uuid.UUID `db:"roomId" json:"roomId"`
	DaysOfWeek []int     `json:"daysOfWeek"`
	StartTime  string    `db:"startTime" json:"startTime"`
	EndTime    string    `db:"endTime" json:"endTime"`
}

type Slot struct {
	Id     uuid.UUID `db:"id" json:"id"`
	RoomId uuid.UUID `db:"roomid" json:"roomId"`
	Start  time.Time `db:"starttime" json:"start"`
	End    time.Time `db:"endtime" json:"end"`
}

type Booking struct {
	Id        uuid.UUID `json:"id"`
	SlotId    uuid.UUID `json:"slotId"`
	UserId    uuid.UUID `json:"userId"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}
