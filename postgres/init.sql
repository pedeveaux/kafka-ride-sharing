CREATE TABLE ride_events (
    id UUID PRIMARY KEY,
    trip_id TEXT NOT NULL,
    event_type VARCHAR(10) NOT NULL,
    event_state VARCHAR(12) NOT NULL,
    event_time TIMESTAMP NOT NULL,
    driver_id TEXT,
    passenger_id TEXT,
    payload JSONB,
    UNIQUE (trip_id, event_type)
);
CREATE INDEX idx_trip_events ON ride_events (trip_id, event_time);
CREATE INDEX idx_event_type ON ride_events (event_type);
CREATE INDEX idx_passenger_id ON ride_events (passenger_id);