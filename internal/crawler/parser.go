package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"emptyclassroom/internal/model"
)

var buildingKeys = []string{
	"building", "buildingName", "buildName", "jxl", "jxlmc", "teachBuilding",
	"teachBuildingName", "teachingBuilding", "teachingBuildingName", "lh", "floorName",
}

var roomKeys = []string{
	"roomNumber", "room_number", "roomNo", "classroomNo", "classroomName",
	"roomName", "room", "jsmc", "jsbh", "fjmc", "name",
}

var occupancyKeys = []string{
	"occupancy", "useStatus", "usingStatus", "status", "occupy", "classStatus",
	"courseStatus", "sectionStatus", "seatStatus", "sjd", "sjzt",
}

type classroomCandidate struct {
	item     map[string]any
	building string
}

func ParseClassrooms(body []byte) ([]model.ClassroomStatus, error) {
	var root any
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse classroom json: %w", err)
	}

	var candidates []classroomCandidate
	collectCandidates(root, "", &candidates)

	seen := make(map[string]struct{})
	rooms := make([]model.ClassroomStatus, 0, len(candidates))
	for _, candidate := range candidates {
		building := firstString(candidate.item, buildingKeys)
		if building == "" {
			building = candidate.building
		}
		roomNumber := firstString(candidate.item, roomKeys)
		occupancy := firstOccupancy(candidate.item)

		if building == "" || roomNumber == "" || occupancy == "" {
			continue
		}
		roomNumber = strings.TrimSpace(strings.TrimPrefix(roomNumber, building))
		if roomNumber == "" {
			continue
		}

		key := building + "\x00" + roomNumber
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		rooms = append(rooms, model.ClassroomStatus{
			Building:   building,
			RoomNumber: roomNumber,
			Occupancy:  occupancy,
		})
	}

	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].Building == rooms[j].Building {
			return rooms[i].RoomNumber < rooms[j].RoomNumber
		}
		return rooms[i].Building < rooms[j].Building
	})

	if len(rooms) == 0 {
		return nil, fmt.Errorf("no classroom records found in response")
	}
	return rooms, nil
}

func collectCandidates(value any, inheritedBuilding string, out *[]classroomCandidate) {
	switch typed := value.(type) {
	case map[string]any:
		building := firstString(typed, buildingKeys)
		if building == "" {
			building = inheritedBuilding
		}
		*out = append(*out, classroomCandidate{item: typed, building: building})
		for _, child := range typed {
			collectCandidates(child, building, out)
		}
	case []any:
		for _, child := range typed {
			collectCandidates(child, inheritedBuilding, out)
		}
	}
}

func firstString(item map[string]any, keys []string) string {
	for _, key := range keys {
		if value, ok := lookup(item, key); ok {
			text := stringify(value)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func firstOccupancy(item map[string]any) string {
	for _, key := range occupancyKeys {
		if value, ok := lookup(item, key); ok {
			if occupancy := normalizeOccupancy(value); occupancy != "" {
				return occupancy
			}
		}
	}
	return occupancyFromSectionKeys(item)
}

func normalizeOccupancy(value any) string {
	switch typed := value.(type) {
	case string:
		var bits []rune
		for _, r := range typed {
			if r == '0' || r == '1' {
				bits = append(bits, r)
			}
		}
		return fitOccupancy(bits)
	case []any:
		bits := make([]rune, 0, len(typed))
		for _, entry := range typed {
			if statusKnown(entry) {
				if busy(entry) {
					bits = append(bits, '1')
				} else {
					bits = append(bits, '0')
				}
			}
		}
		return fitOccupancy(bits)
	default:
		if statusKnown(typed) {
			if busy(typed) {
				return "100000000000"
			}
			return "000000000000"
		}
		return ""
	}
}

func occupancyFromSectionKeys(item map[string]any) string {
	prefixes := []string{"section", "lesson", "period", "jc", "c", "p", "time", "s", ""}
	bits := make([]rune, model.SlotCount)
	found := false
	for i := 1; i <= model.SlotCount; i++ {
		bits[i-1] = '0'
		for _, prefix := range prefixes {
			key := prefix + strconv.Itoa(i)
			if value, ok := lookup(item, key); ok && statusKnown(value) {
				found = true
				if busy(value) {
					bits[i-1] = '1'
				}
				break
			}
		}
	}
	if !found {
		return ""
	}
	return string(bits)
}

func fitOccupancy(bits []rune) string {
	if len(bits) == 0 {
		return ""
	}
	if len(bits) > model.SlotCount {
		bits = bits[:model.SlotCount]
	}
	for len(bits) < model.SlotCount {
		bits = append(bits, '0')
	}
	return string(bits)
}

func lookup(item map[string]any, key string) (any, bool) {
	normalized := normalizeKey(key)
	for candidate, value := range item {
		if normalizeKey(candidate) == normalized {
			return value, true
		}
	}
	return nil, false
}

func normalizeKey(key string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return strings.ToLower(replacer.Replace(key))
}

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(typed, 'f', 2, 64), "0"), ".")
	case int:
		return strconv.Itoa(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func statusKnown(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool, float64, int:
		return true
	case string:
		text := strings.TrimSpace(strings.ToLower(typed))
		return text != "" && text != "-" && text != "null"
	default:
		return true
	}
}

func busy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case string:
		text := strings.TrimSpace(strings.ToLower(typed))
		if text == "" || text == "0" || text == "false" || text == "free" || text == "available" || containsAny(text, "\u7a7a", "\u95f2") {
			return false
		}
		if text == "1" || text == "true" || text == "busy" || text == "used" || containsAny(text, "\u5360", "\u6ee1", "\u5df2\u7528") {
			return true
		}
		parsed, err := strconv.ParseFloat(text, 64)
		return err == nil && parsed != 0
	default:
		return false
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
