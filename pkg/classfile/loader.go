package classfile

import (
	"fmt"
	"java-app/pkg/jar"
	"log"
	"strings"
	"sync"
)

type ClassLoader struct {
	Archive *jar.JarArchive
	mu      sync.RWMutex
	classes map[string]*ClassFile
}

func NewClassLoader(archive *jar.JarArchive) *ClassLoader {
	return &ClassLoader{
		Archive: archive,
		classes: make(map[string]*ClassFile),
	}
}

func (cl *ClassLoader) LoadClass(name string) (*ClassFile, error) {
	cl.mu.RLock()
	if cf, ok := cl.classes[name]; ok {
		cl.mu.RUnlock()
		return cf, nil
	}
	cl.mu.RUnlock()

	if cl.Archive == nil {
		return nil, fmt.Errorf("class %s not found in JAR archive", name)
	}
	classBytes, ok := cl.Archive.Classes[name]
	if !ok {
		return nil, fmt.Errorf("class %s not found in JAR archive", name)
	}

	cf, err := Parse(classBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse class %s: %w", name, err)
	}

	cl.mu.Lock()
	defer cl.mu.Unlock()
	if existing, ok := cl.classes[name]; ok {
		return existing, nil
	}
	if cf != nil {
		cl.classes[name] = cf
		if strings.HasSuffix(name, "bl") || strings.HasSuffix(name, "fu") || strings.HasSuffix(name, "bx") {
			for _, m := range cf.Methods {
				log.Printf("[CLASS-INSPECT] %s method: %s%s", name, cf.ConstantPool.GetUtf8(m.NameIndex), cf.ConstantPool.GetUtf8(m.DescIndex))
			}
		}
	}
	return cf, nil
}
