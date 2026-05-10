package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"emptyclassroom/internal/model"
)

type Repository interface {
	UpsertClassrooms(ctx context.Context, campusID int, date time.Time, rooms []model.ClassroomStatus) error
	ListClassrooms(ctx context.Context, campusID int, date time.Time) ([]model.ClassroomStatus, error)
}

type Crawler interface {
	Login(ctx context.Context) error
	FetchToday(ctx context.Context, campusID int) ([]model.ClassroomStatus, error)
}

type ClassroomService struct {
	repo    Repository
	crawler Crawler
	loc     *time.Location
	cache   *summaryCache
}

func NewClassroomService(repo Repository, crawler Crawler, loc *time.Location) *ClassroomService {
	return &ClassroomService{
		repo:    repo,
		crawler: crawler,
		loc:     loc,
		cache:   newSummaryCache(90 * time.Second),
	}
}

func (s *ClassroomService) Campuses() []model.Campus {
	out := make([]model.Campus, len(model.Campuses))
	copy(out, model.Campuses)
	return out
}

func (s *ClassroomService) Slots() []model.Slot {
	out := make([]model.Slot, len(model.Slots))
	copy(out, model.Slots)
	return out
}

func (s *ClassroomService) List(ctx context.Context, campusID int, date time.Time, slots []int) (model.ClassroomSummary, error) {
	if date.IsZero() {
		date = time.Now().In(s.loc)
	}
	date = dateInLocation(date, s.loc)
	current := currentSlot(time.Now().In(s.loc))
	selected := normalizeSlots(slots, current)

	if summary, ok := s.cache.Get(campusID, date, selected); ok {
		return summary, nil
	}

	rooms, err := s.repo.ListClassrooms(ctx, campusID, date)
	if err != nil {
		return model.ClassroomSummary{}, err
	}
	summary := buildSummary(campusID, date, current, selected, rooms)
	s.cache.Set(campusID, date, selected, summary)
	return summary, nil
}

func (s *ClassroomService) SyncCampusToday(ctx context.Context, campusID int) error {
	if s.crawler == nil {
		return fmt.Errorf("crawler is not configured")
	}
	rooms, err := s.crawler.FetchToday(ctx, campusID)
	if err != nil {
		return err
	}
	date := dateInLocation(time.Now().In(s.loc), s.loc)
	for i := range rooms {
		rooms[i].CampusID = campusID
		rooms[i].Date = date
	}
	if err := s.repo.UpsertClassrooms(ctx, campusID, date, rooms); err != nil {
		return err
	}
	s.cache.Delete(campusID, date)
	return nil
}

func (s *ClassroomService) SyncAllToday(ctx context.Context, campusIDs []int) error {
	if len(campusIDs) == 0 {
		return nil
	}

	if err := s.crawler.Login(ctx); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	var errs []error
	for _, campusID := range campusIDs {
		if err := s.SyncCampusToday(ctx, campusID); err != nil {
			errs = append(errs, fmt.Errorf("campus %d: %w", campusID, err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return joinErrors(errs)
}

func buildSummary(campusID int, date time.Time, currentSlotValue int, selectedSlots []int, rooms []model.ClassroomStatus) model.ClassroomSummary {
	groups := make(map[string][]model.RoomStatus)
	for _, room := range rooms {
		status := model.RoomStatus{
			ID:         room.ID,
			CampusID:   room.CampusID,
			Building:   room.Building,
			RoomNumber: room.RoomNumber,
			Occupancy:  room.Occupancy,
			FreeNow:    freeAtSlots(room.Occupancy, selectedSlots),
		}
		groups[room.Building] = append(groups[room.Building], status)
	}

	buildings := make([]model.BuildingSummary, 0, len(groups))
	for building, groupedRooms := range groups {
		sort.Slice(groupedRooms, func(i, j int) bool {
			return groupedRooms[i].RoomNumber < groupedRooms[j].RoomNumber
		})

		freeRooms := 0
		for _, room := range groupedRooms {
			if room.FreeNow {
				freeRooms++
			}
		}
		ratio := 0.0
		if len(groupedRooms) > 0 {
			ratio = math.Round(float64(freeRooms)/float64(len(groupedRooms))*1000) / 10
		}
		buildings = append(buildings, model.BuildingSummary{
			Building:   building,
			TotalRooms: len(groupedRooms),
			FreeRooms:  freeRooms,
			FreeRatio:  ratio,
			Rooms:      groupedRooms,
		})
	}

	sort.Slice(buildings, func(i, j int) bool {
		return buildings[i].Building < buildings[j].Building
	})

	return model.ClassroomSummary{
		CampusID:      campusID,
		Date:          date.Format("2006-01-02"),
		CurrentSlot:   currentSlotValue,
		SelectedSlots: selectedSlots,
		Buildings:     buildings,
	}
}

func freeAtSlots(occupancy string, slots []int) bool {
	if len(slots) == 0 {
		for _, r := range occupancy {
			if r == '1' {
				return false
			}
		}
		return true
	}
	for _, slot := range slots {
		if slot < 1 || slot > model.SlotCount {
			continue
		}
		if len(occupancy) < slot {
			continue
		}
		if occupancy[slot-1] == '1' {
			return false
		}
	}
	return true
}

func currentSlot(now time.Time) int {
	for _, item := range model.Slots {
		start := parseClock(now, item.Start)
		end := parseClock(now, item.End)
		if (now.Equal(start) || now.After(start)) && now.Before(end) {
			return item.Number
		}
		if now.Before(start) {
			return item.Number
		}
	}
	return 0
}

func normalizeSlots(requested []int, current int) []int {
	if len(requested) > 0 {
		valid := make([]int, 0, len(requested))
		for _, s := range requested {
			if s >= 1 && s <= model.SlotCount {
				valid = append(valid, s)
			}
		}
		if len(valid) > 0 {
			sort.Ints(valid)
			return valid
		}
	}
	if current >= 1 && current <= model.SlotCount {
		return []int{current}
	}
	return []int{1}
}

func parseClock(base time.Time, value string) time.Time {
	parts := strings.SplitN(value, ":", 2)
	hour, _ := strconv.Atoi(parts[0])
	minute := 0
	if len(parts) == 2 {
		minute, _ = strconv.Atoi(parts[1])
	}
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location())
}

func dateInLocation(value time.Time, loc *time.Location) time.Time {
	inLoc := value.In(loc)
	return time.Date(inLoc.Year(), inLoc.Month(), inLoc.Day(), 0, 0, 0, 0, loc)
}

func joinErrors(errs []error) error {
	var builder strings.Builder
	for i, err := range errs {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(err.Error())
	}
	return errors.New(builder.String())
}
