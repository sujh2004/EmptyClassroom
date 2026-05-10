package crawler

import "testing"

func TestParseClassrooms(t *testing.T) {
	// Real BUPT API format: each slot lists occupied classrooms as "building-room(capacity)"
	body := []byte(`{"code":"1","Msg":"success","data":[
		{"CLASSROOMS":"\u6559\u4e09-101(60),\u6559\u4e09-102(80)","NODENAME":"1","NODETIME":"08:00-08:45"},
		{"CLASSROOMS":"\u6559\u4e09-101(60)","NODENAME":"2","NODETIME":"08:50-09:35"}
	]}`)
	rooms, err := ParseClassrooms(body)
	if err != nil {
		t.Fatalf("parse classrooms: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("len(rooms) = %d, want 2", len(rooms))
	}
	// 教三-101: occupied slot 1+2 → "11000000000000"
	// 教三-102: occupied slot 1 only → "10000000000000"
	for _, r := range rooms {
		if r.Building == "教三" && r.RoomNumber == "101" {
			if r.Occupancy != "11000000000000" {
				t.Fatalf("教三-101 occupancy = %s, want 11000000000000", r.Occupancy)
			}
		} else if r.Building == "教三" && r.RoomNumber == "102" {
			if r.Occupancy != "10000000000000" {
				t.Fatalf("教三-102 occupancy = %s, want 10000000000000", r.Occupancy)
			}
		} else {
			t.Fatalf("unexpected room: %s-%s", r.Building, r.RoomNumber)
		}
	}
}

func TestParseClassroomsEmpty(t *testing.T) {
	// Weekend / no data: CLASSROOMS is an object, not a string
	body := []byte(`{"code":"1","Msg":"success","data":[{"CLASSROOMS":{"array":false},"NODENAME":"1"}]}`)
	_, err := ParseClassrooms(body)
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}
