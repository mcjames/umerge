package ui

import (
	"os"
	"path/filepath"
	"testing"

	"umerge/internal/entry"
)

func strptr(s string) *string { return &s }

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAllSidesPresent_TwoWay(t *testing.T) {
	cases := []struct {
		name string
		e    *entry.Entry
		want bool
	}{
		{"both present", &entry.Entry{Left: strptr("/a"), Right: strptr("/b")}, true},
		{"left only", &entry.Entry{Left: strptr("/a")}, false},
		{"right only", &entry.Entry{Right: strptr("/b")}, false},
		{"neither", &entry.Entry{}, false},
	}
	for _, tc := range cases {
		if got := allSidesPresent(tc.e, 2); got != tc.want {
			t.Errorf("%s: allSidesPresent(ways=2) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestAllSidesPresent_ThreeWay(t *testing.T) {
	cases := []struct {
		name string
		e    *entry.Entry
		want bool
	}{
		{"all three present", &entry.Entry{Left: strptr("/a"), Middle: strptr("/m"), Right: strptr("/b")}, true},
		{"missing middle", &entry.Entry{Left: strptr("/a"), Right: strptr("/b")}, false},
		{"missing left", &entry.Entry{Middle: strptr("/m"), Right: strptr("/b")}, false},
		{"missing right", &entry.Entry{Left: strptr("/a"), Middle: strptr("/m")}, false},
	}
	for _, tc := range cases {
		if got := allSidesPresent(tc.e, 3); got != tc.want {
			t.Errorf("%s: allSidesPresent(ways=3) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCompareEntry_TwoWaySame(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")
	e := &entry.Entry{Left: &left, Right: &right}

	msg := compareEntry(e, 2)
	if msg.state != entry.Same {
		t.Errorf("state = %v, want Same", msg.state)
	}
	if msg.numDiffs != 0 {
		t.Errorf("numDiffs = %d, want 0", msg.numDiffs)
	}
}

func TestCompareEntry_TwoWayDifferent(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "one\n")
	right := writeFile(t, dir, "right.txt", "two\n")
	e := &entry.Entry{Left: &left, Right: &right}

	msg := compareEntry(e, 2)
	if msg.state != entry.Different {
		t.Errorf("state = %v, want Different", msg.state)
	}
	if msg.numDiffs == 0 {
		t.Errorf("numDiffs = %d, want > 0", msg.numDiffs)
	}
}

func TestCompareEntry_TwoWayError(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "one\n")
	missing := filepath.Join(dir, "does-not-exist.txt")
	e := &entry.Entry{Left: &left, Right: &missing}

	msg := compareEntry(e, 2)
	if msg.state != entry.CompareError {
		t.Errorf("state = %v, want CompareError", msg.state)
	}
}

func TestCompareEntry_ThreeWaySame(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeFile(t, dir, "right.txt", "same\n")
	e := &entry.Entry{Left: &left, Middle: &middle, Right: &right}

	msg := compareEntry(e, 3)
	if msg.state != entry.Same {
		t.Errorf("state = %v, want Same", msg.state)
	}
	if msg.lmDiffs != 0 || msg.mrDiffs != 0 {
		t.Errorf("lmDiffs=%d mrDiffs=%d, want 0,0", msg.lmDiffs, msg.mrDiffs)
	}
}

func TestCompareEntry_ThreeWayDifferent(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "AAA\n")
	middle := writeFile(t, dir, "middle.txt", "BBB\n")
	right := writeFile(t, dir, "right.txt", "CCC\n")
	e := &entry.Entry{Left: &left, Middle: &middle, Right: &right}

	msg := compareEntry(e, 3)
	if msg.state != entry.Different {
		t.Errorf("state = %v, want Different", msg.state)
	}
	if msg.lmDiffs == 0 || msg.mrDiffs == 0 {
		t.Errorf("lmDiffs=%d mrDiffs=%d, want both > 0", msg.lmDiffs, msg.mrDiffs)
	}
}

func TestCompareEntry_ThreeWayError(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "AAA\n")
	middle := writeFile(t, dir, "middle.txt", "BBB\n")
	missing := filepath.Join(dir, "does-not-exist.txt")
	e := &entry.Entry{Left: &left, Middle: &middle, Right: &missing}

	msg := compareEntry(e, 3)
	if msg.state != entry.CompareError {
		t.Errorf("state = %v, want CompareError", msg.state)
	}
}

func writeBinaryFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCompareEntry_TwoWayBinaryDifferent(t *testing.T) {
	dir := t.TempDir()
	left := writeBinaryFile(t, dir, "left.bin", []byte{0x00, 0x01, 0x02})
	right := writeBinaryFile(t, dir, "right.bin", []byte{0x00, 0xFF, 0xFE})
	e := &entry.Entry{Left: &left, Right: &right}

	msg := compareEntry(e, 2)
	if msg.state != entry.BinaryDifferent {
		t.Errorf("state = %v, want BinaryDifferent", msg.state)
	}
}

func TestCompareEntry_ThreeWayBinaryDifferent(t *testing.T) {
	dir := t.TempDir()
	left := writeFile(t, dir, "left.txt", "same\n")
	middle := writeFile(t, dir, "middle.txt", "same\n")
	right := writeBinaryFile(t, dir, "right.bin", []byte{0x00, 0x01, 0x02})
	e := &entry.Entry{Left: &left, Middle: &middle, Right: &right}

	msg := compareEntry(e, 3)
	if msg.state != entry.BinaryDifferent {
		t.Errorf("state = %v, want BinaryDifferent", msg.state)
	}
}
