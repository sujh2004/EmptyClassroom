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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
