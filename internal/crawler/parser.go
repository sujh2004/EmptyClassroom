package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"emptyclassroom/internal/model"
)

type jwSlotEntry struct {
	Classrooms any    `json:"CLASSROOMS"`
	NodeName   string `json:"NODENAME"`
}

type jwResponse struct {
	Code string        `json:"code"`
	Msg  string        `json:"Msg"`
	Data []jwSlotEntry `json:"data"`
}

func ParseClassrooms(body []byte) ([]model.ClassroomStatus, error) {
	var resp jwResponse
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("parse classroom json: %w", err)
	}

	if resp.Code != "1" {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	type roomKey struct {
		building   string
		roomNumber string
	}

	occupancy := make(map[roomKey][]byte)

	for _, entry := range resp.Data {
		classroomsStr, ok := entry.Classrooms.(string)
		if !ok || classroomsStr == "" {
			continue
		}

		slot := 0
		if _, err := fmt.Sscanf(entry.NodeName, "%d", &slot); err != nil || slot < 1 || slot > model.SlotCount {
			continue
		}

		items := strings.Split(classroomsStr, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}

			// Format: "楼名-教室号(容量)" e.g. "教一-101(60)"
			name := strings.Split(item, "(")[0]
			building, room, found := strings.Cut(name, "-")
			if !found || building == "" || room == "" {
				continue
			}

			key := roomKey{building: building, roomNumber: room}
			if _, exists := occupancy[key]; !exists {
				bits := make([]byte, model.SlotCount)
				for i := range bits {
					bits[i] = '0'
				}
				occupancy[key] = bits
			}
			occupancy[key][slot-1] = '1'
		}
	}

	if len(occupancy) == 0 {
		return nil, fmt.Errorf("no classroom records found in response")
	}

	rooms := make([]model.ClassroomStatus, 0, len(occupancy))
	for key, bits := range occupancy {
		rooms = append(rooms, model.ClassroomStatus{
			Building:   key.building,
			RoomNumber: key.roomNumber,
			Occupancy:  string(bits),
		})
	}

	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].Building == rooms[j].Building {
			return rooms[i].RoomNumber < rooms[j].RoomNumber
		}
		return rooms[i].Building < rooms[j].Building
	})

	return rooms, nil
}
