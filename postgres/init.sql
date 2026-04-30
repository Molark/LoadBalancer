CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(50) UNIQUE NOT NULL,
    passwordHash VARCHAR(255) NOT NULL DEFAULT '',
    role VARCHAR(5) NOT NULL CHECK (role IN ('admin', 'user')),
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS rooms (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(80) UNIQUE NOT NULL,
    description VARCHAR(255),
    capacity integer,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS schedules (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    roomId UUID NOT NULL REFERENCES rooms(id),
    startTime TIME NOT NULL,
    endTime TIME NOT NULL
    );

CREATE TABLE IF NOT EXISTS slots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    roomId UUID NOT NULL REFERENCES rooms(id),
    startTime TIMESTAMP NOT NULL,
    endTime TIMESTAMP NOT NULL,
    booked BOOLEAN DEFAULT FALSE,

    CONSTRAINT check_duration
    CHECK (endTime - startTime = INTERVAL '30 minutes')
    );

CREATE TABLE IF NOT EXISTS bookings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    slotId UUID NOT NULL REFERENCES slots(id),
    userId UUID NOT NULL REFERENCES users(id),
    status VARCHAR(9) CHECK (status IN ('active', 'cancelled')),
    conferenceLink VARCHAR(50),
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS daysOfWeek (
    id int PRIMARY KEY CHECK (id > 0 AND id < 8),
    name VARCHAR(20)
    );

CREATE TABLE IF NOT EXISTS schedule_weekdays (
    scheduleId UUID REFERENCES schedules(id),
    weekDayId integer NOT NULL REFERENCES daysOfWeek(id) CHECK (weekDayId > 0 AND weekDayId < 8),
    PRIMARY KEY (scheduleId, weekDayId)
    );

INSERT INTO daysOfWeek (id, name) VALUES
                                      (1, 'Monday'),
                                      (2, 'Tuesday'),
                                      (3, 'Wednesday'),
                                      (4, 'Thursday'),
                                      (5, 'Friday'),
                                      (6, 'Saturday'),
                                      (7, 'Sunday')
    ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, role)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@dummy.com', 'admin'),
    ('00000000-0000-0000-0000-000000000002', 'user@edummy.com',   'user')
    ON CONFLICT (id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_slots_room_start_booked
    ON slots (roomid, starttime, booked);