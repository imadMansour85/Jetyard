package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestPaginateNoParamsReturnsEverything(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	r := httptest.NewRequest("GET", "/api/streams", nil)

	got := paginate(r, items)
	if len(got) != len(items) {
		t.Fatalf("paginate with no params = %v, want all %v", got, items)
	}
}

func TestPaginateLimit(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	r := httptest.NewRequest("GET", "/api/streams?limit=2", nil)

	got := paginate(r, items)
	want := []int{1, 2}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("paginate with limit=2 = %v, want %v", got, want)
	}
}

func TestPaginateOffset(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	r := httptest.NewRequest("GET", "/api/streams?offset=3", nil)

	got := paginate(r, items)
	want := []int{4, 5}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("paginate with offset=3 = %v, want %v", got, want)
	}
}

func TestPaginateLimitAndOffset(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	r := httptest.NewRequest("GET", "/api/streams?offset=1&limit=2", nil)

	got := paginate(r, items)
	want := []int{2, 3}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("paginate with offset=1&limit=2 = %v, want %v", got, want)
	}
}

func TestPaginateOffsetPastEndReturnsEmpty(t *testing.T) {
	items := []int{1, 2, 3}
	r := httptest.NewRequest("GET", "/api/streams?offset=10", nil)

	got := paginate(r, items)
	if len(got) != 0 {
		t.Fatalf("paginate with offset past end = %v, want empty", got)
	}
}
