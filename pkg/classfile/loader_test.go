package classfile

import (
	"os"
	"sync"
	"testing"

	"java-app/pkg/jar"
)

func TestLoadClassConcurrent(t *testing.T) {
	data, err := os.ReadFile("../../mobile/assets/KnightOnline_402.jar")
	if err != nil {
		t.Skip(err)
	}
	archive, err := jar.ReadJar(data)
	if err != nil {
		t.Fatal(err)
	}
	cl := NewClassLoader(archive)
	names := []string{"fu", "p", "u", "cg", "bx", "bl", "com/silverknight/a"}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, name := range names {
				if _, err := cl.LoadClass(name); err != nil {
					t.Errorf("LoadClass(%s): %v", name, err)
				}
			}
		}()
	}
	wg.Wait()
}
