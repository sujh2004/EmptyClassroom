package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"emptyclassroom/internal/model"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQL(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Migrate(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `ALTER TABLE classroom_status MODIFY COLUMN occupancy CHAR(14) NOT NULL`)
	return err
}

func (r *MySQLRepository) UpsertClassrooms(ctx context.Context, campusID int, date time.Time, rooms []model.ClassroomStatus) error {
	if len(rooms) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO classroom_status (campus_id, building, room_number, occupancy, date)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  occupancy = VALUES(occupancy),
  updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	dateValue := date.Format("2006-01-02")
	for _, room := range rooms {
		if room.Building == "" || room.RoomNumber == "" || room.Occupancy == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, campusID, room.Building, room.RoomNumber, room.Occupancy, dateValue); err != nil {
			return fmt.Errorf("upsert classroom %s %s: %w", room.Building, room.RoomNumber, err)
		}
	}

	return tx.Commit()
}

func (r *MySQLRepository) ListClassrooms(ctx context.Context, campusID int, date time.Time) ([]model.ClassroomStatus, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, campus_id, building, room_number, occupancy, date, updated_at
FROM classroom_status
WHERE campus_id = ? AND date = ?
ORDER BY building ASC, room_number ASC`, campusID, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []model.ClassroomStatus
	for rows.Next() {
		var room model.ClassroomStatus
		if err := rows.Scan(&room.ID, &room.CampusID, &room.Building, &room.RoomNumber, &room.Occupancy, &room.Date, &room.UpdatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *MySQLRepository) LatestDate(ctx context.Context, campusID int) (time.Time, error) {
	var date sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT MAX(date) FROM classroom_status WHERE campus_id = ?`, campusID).Scan(&date)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil || !date.Valid {
		return time.Time{}, err
	}
	return date.Time, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS classroom_status (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  campus_id TINYINT NOT NULL DEFAULT 0,
  building VARCHAR(100) NOT NULL,
  room_number VARCHAR(100) NOT NULL,
  occupancy CHAR(14) NOT NULL,
  date DATE NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_classroom_day (campus_id, date, building, room_number),
  KEY idx_date_campus (date, campus_id),
  KEY idx_building (building)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
