package jar

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
)

type JarArchive struct {
	Manifest        map[string]string
	MainMIDletClass string
	Classes         map[string][]byte
	Resources       map[string][]byte
}

func parseManifest(data []byte) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			result[key] = val
		}
	}
	return result
}

func extractMainClass(midlet1Val string) string {
	parts := strings.Split(midlet1Val, ",")
	if len(parts) >= 3 {
		cls := strings.TrimSpace(parts[2])
		return strings.ReplaceAll(cls, ".", "/")
	}
	return ""
}

func ReadJar(jarBytes []byte) (*JarArchive, error) {
	reader, err := zip.NewReader(bytes.NewReader(jarBytes), int64(len(jarBytes)))
	if err != nil {
		return nil, err
	}

	archive := &JarArchive{
		Manifest:  make(map[string]string),
		Classes:   make(map[string][]byte),
		Resources: make(map[string][]byte),
	}

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		if file.Name == "META-INF/MANIFEST.MF" {
			archive.Manifest = parseManifest(data)
			if val, ok := archive.Manifest["MIDlet-1"]; ok {
				archive.MainMIDletClass = extractMainClass(val)
			}
		} else if strings.HasSuffix(file.Name, ".class") {
			className := strings.TrimSuffix(file.Name, ".class")
			archive.Classes[className] = data
		} else {
			cleanName := strings.TrimPrefix(file.Name, "/")
			archive.Resources[cleanName] = data
		}
	}

	return archive, nil
}
