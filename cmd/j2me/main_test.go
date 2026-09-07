package main

import "testing"

func TestAppTitle(t *testing.T) {
	expected := "J2ME Go Emulator"
	if AppTitle != expected {
		t.Errorf("expected %s, got %s", expected, AppTitle)
	}
}
