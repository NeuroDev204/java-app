package jar

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseManifest(t *testing.T) {
	manifestContent := "MIDlet-Name: TestGame\r\nMIDlet-1: TestGame,,com.vendor.TestMIDlet\r\nMIDlet-Vendor: Vendor\r\n"
	manifestMap := parseManifest([]byte(manifestContent))

	if manifestMap["MIDlet-Name"] != "TestGame" {
		t.Errorf("expected MIDlet-Name to be TestGame, got %s", manifestMap["MIDlet-Name"])
	}
	if manifestMap["MIDlet-1"] != "TestGame,,com.vendor.TestMIDlet" {
		t.Errorf("unexpected MIDlet-1: %s", manifestMap["MIDlet-1"])
	}

	mainClass := extractMainClass(manifestMap["MIDlet-1"])
	if mainClass != "com/vendor/TestMIDlet" {
		t.Errorf("expected main class com/vendor/TestMIDlet, got %s", mainClass)
	}
}

func TestReadJar(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	manifestData := []byte("MIDlet-Name: KnightOnline\nMIDlet-1: KnightOnline,icon.png,com.j2me.KnightMIDlet\n")
	mw, err := zw.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatalf("failed to create manifest in zip: %v", err)
	}
	mw.Write(manifestData)

	cw, err := zw.Create("com/j2me/KnightMIDlet.class")
	if err != nil {
		t.Fatalf("failed to create class file in zip: %v", err)
	}
	classBytes := []byte{0xCA, 0xFE, 0xBA, 0xBE, 0x00, 0x00, 0x00, 0x32}
	cw.Write(classBytes)

	rw, err := zw.Create("assets/logo.png")
	if err != nil {
		t.Fatalf("failed to create resource in zip: %v", err)
	}
	rw.Write([]byte("fake-png-data"))

	zw.Close()

	archive, err := ReadJar(buf.Bytes())
	if err != nil {
		t.Fatalf("ReadJar failed: %v", err)
	}

	if archive.MainMIDletClass != "com/j2me/KnightMIDlet" {
		t.Errorf("expected main class com/j2me/KnightMIDlet, got %s", archive.MainMIDletClass)
	}
	if !bytes.Equal(archive.Classes["com/j2me/KnightMIDlet"], classBytes) {
		t.Errorf("class byte content mismatch")
	}
	if !bytes.Equal(archive.Resources["assets/logo.png"], []byte("fake-png-data")) {
		t.Errorf("resource byte content mismatch")
	}
}

func TestReadJarInvalidData(t *testing.T) {
	_, err := ReadJar([]byte("invalid zip content"))
	if err == nil {
		t.Errorf("expected error reading invalid JAR zip bytes, got nil")
	}
}

