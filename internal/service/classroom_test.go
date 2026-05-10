package service

import (
	"testing"
	"time"
)

func TestCurrentSlotUsesFourteenSlotSchedule(t *testing.T) {
	loc := time.FixedZone("test", 8*60*60)
	tests := []struct {
		clock string
		want  int
	}{
		{clock: "08:00", want: 1},
		{clock: "12:30", want: 6},
		{clock: "13:44", want: 6},
		{clock: "13:50", want: 7},
		{clock: "20:10", want: 14},
		{clock: "21:00", want: 0},
	}

	for _, tt := range tests {
		now, err := time.ParseInLocation("2006-01-02 15:04", "2026-05-06 "+tt.clock, loc)
		if err != nil {
			t.Fatalf("parse time: %v", err)
		}
		if got := currentSlot(now); got != tt.want {
			t.Fatalf("currentSlot(%s) = %d, want %d", tt.clock, got, tt.want)
		}
	}
}

func TestFreeAtSlotsMultiple(t *testing.T) {
	occupancy := "01010000000000"

	if freeAtSlots(occupancy, []int{1}) != true {
		t.Fatal("slot 1 should be free")
	}
	if freeAtSlots(occupancy, []int{2}) != false {
		t.Fatal("slot 2 should be busy")
	}
	if freeAtSlots(occupancy, []int{1, 3}) != true {
		t.Fatal("slots 1,3 should both be free")
	}
	if freeAtSlots(occupancy, []int{1, 2}) != false {
		t.Fatal("slots 1,2 should not both be free (slot 2 is busy)")
	}
	if freeAtSlots(occupancy, []int{5, 6, 7}) != true {
		t.Fatal("slots 5,6,7 should all be free")
	}
}

func TestNormalizeSlots(t *testing.T) {
	if got := normalizeSlots([]int{3, 1, 2}, 5); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("normalizeSlots([3,1,2], 5) = %v, want [1,2,3]", got)
	}
	if got := normalizeSlots(nil, 5); len(got) != 1 || got[0] != 5 {
		t.Fatalf("normalizeSlots(nil, 5) = %v, want [5]", got)
	}
	if got := normalizeSlots(nil, 0); len(got) != 1 || got[0] != 1 {
		t.Fatalf("normalizeSlots(nil, 0) = %v, want [1]", got)
	}
}
