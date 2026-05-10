package crawler

import "testing"

func TestParseClassroomsFlatRecords(t *testing.T) {
	body := []byte(`{"data":[{"building":"\u6559\u4e09\u697c","roomNumber":"101","occupancy":"01010101010101"}]}`)
	rooms, err := ParseClassrooms(body)
	if err != nil {
		t.Fatalf("parse classrooms: %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("len(rooms) = %d, want 1", len(rooms))
	}
	if rooms[0].Building != "\u6559\u4e09\u697c" || rooms[0].RoomNumber != "101" || rooms[0].Occupancy != "01010101010101" {
		t.Fatalf("room = %+v", rooms[0])
	}
}

func TestParseClassroomsNestedBuilding(t *testing.T) {
	body := []byte(`{"data":[{"buildingName":"\u6559\u5b66\u697cA","rooms":[{"roomNo":"201","status":[0,1,0,1,0,1,0,1,0,1,0,1,0,1]}]}]}`)
	rooms, err := ParseClassrooms(body)
	if err != nil {
		t.Fatalf("parse classrooms: %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("len(rooms) = %d, want 1", len(rooms))
	}
	if rooms[0].Building != "\u6559\u5b66\u697cA" || rooms[0].RoomNumber != "201" || rooms[0].Occupancy != "01010101010101" {
		t.Fatalf("room = %+v", rooms[0])
	}
}
