package model

import "time"

type Campus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

const SlotCount = 14

type Slot struct {
	Number int    `json:"number"`
	Start  string `json:"start"`
	End    string `json:"end"`
}

type ClassroomStatus struct {
	ID         int64     `json:"id"`
	CampusID   int       `json:"campus_id"`
	Building   string    `json:"building"`
	RoomNumber string    `json:"room_number"`
	Occupancy  string    `json:"occupancy"`
	Date       time.Time `json:"-"`
	UpdatedAt  time.Time `json:"-"`
}

type RoomStatus struct {
	ID         int64  `json:"id"`
	CampusID   int    `json:"campus_id"`
	Building   string `json:"building"`
	RoomNumber string `json:"room_number"`
	Occupancy  string `json:"occupancy"`
	FreeNow    bool   `json:"free_now"`
}

type BuildingSummary struct {
	Building   string       `json:"building"`
	TotalRooms int          `json:"total_rooms"`
	FreeRooms  int          `json:"free_rooms"`
	FreeRatio  float64      `json:"free_ratio"`
	Rooms      []RoomStatus `json:"rooms"`
}

type ClassroomSummary struct {
	CampusID      int               `json:"campus_id"`
	Date          string            `json:"date"`
	CurrentSlot   int               `json:"current_slot"`
	SelectedSlots []int             `json:"selected_slots"`
	Buildings     []BuildingSummary `json:"buildings"`
}

var Campuses = []Campus{
	{ID: 0, Name: "西土城"},
	{ID: 1, Name: "沙河"},
}

var Slots = []Slot{
	{Number: 1, Start: "08:00", End: "08:45"},
	{Number: 2, Start: "08:50", End: "09:35"},
	{Number: 3, Start: "09:50", End: "10:35"},
	{Number: 4, Start: "10:40", End: "11:25"},
	{Number: 5, Start: "11:30", End: "12:15"},
	{Number: 6, Start: "13:00", End: "13:45"},
	{Number: 7, Start: "13:50", End: "14:35"},
	{Number: 8, Start: "14:45", End: "15:30"},
	{Number: 9, Start: "15:40", End: "16:25"},
	{Number: 10, Start: "16:35", End: "17:20"},
	{Number: 11, Start: "17:25", End: "18:10"},
	{Number: 12, Start: "18:30", End: "19:15"},
	{Number: 13, Start: "19:20", End: "20:05"},
	{Number: 14, Start: "20:10", End: "20:55"},
}
