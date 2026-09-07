package jvm

/*
#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux
#cgo LDFLAGS: -L/usr/lib/jvm/java-21-openjdk-amd64/lib/server -Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server -ljvm

#include <jni.h>
#include <stdlib.h>
#include <string.h>

// create_java_vm initializes a HotSpot JVM via JNI_CreateJavaVM.
static jint create_java_vm(JavaVM **p_vm, JNIEnv **p_env, const char *classpath_opt) {
	JavaVMInitArgs vm_args;
	JavaVMOption options[2];

	options[0].optionString = (char *)classpath_opt;
	options[0].extraInfo = NULL;
	options[1].optionString = (char *)"-Xrs";
	options[1].extraInfo = NULL;

	memset(&vm_args, 0, sizeof(vm_args));
	vm_args.version = JNI_VERSION_1_8;
	vm_args.nOptions = 2;
	vm_args.options = options;
	vm_args.ignoreUnrecognized = JNI_FALSE;

	return JNI_CreateJavaVM(p_vm, (void**)p_env, &vm_args);
}

// destroy_java_vm tears down the JVM instance.
static void destroy_java_vm(JavaVM *vm) {
	if (vm != NULL && *vm != NULL) {
		(*vm)->DestroyJavaVM(vm);
	}
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"
)

// JVM wraps the native C JNI JavaVM and JNIEnv pointers.
type JVM struct {
	vm  *C.JavaVM
	env *C.JNIEnv
}

var (
	globalJVM *JVM
	jvmMutex  sync.Mutex
)

// GetGlobalJVM returns the initialized HotSpot JVM instance, or nil if not initialized.
func GetGlobalJVM() *JVM {
	return globalJVM
}

// InitJVM initializes an embedded JVM using CGO and JNI.
// If classpath is empty, it defaults to "-Djava.class.path=./j2me-stubs/bin".
// HotSpot allows only one JVM instance per OS process, so subsequent calls return
// the existing global JVM instance.
func InitJVM(classpath string) (*JVM, error) {
	// Lock OS thread before making JNI calls to ensure JNIEnv thread-local stability.
	runtime.LockOSThread()

	jvmMutex.Lock()
	defer jvmMutex.Unlock()

	if globalJVM != nil {
		return globalJVM, nil
	}

	if classpath == "" {
		classpath = "./j2me-stubs/bin"
	}

	cpPath := classpath
	if len(cpPath) >= 17 && cpPath[:17] == "-Djava.class.path=" {
		cpPath = cpPath[17:]
	}

	candidates := []string{cpPath, "./j2me-stubs/bin", "../j2me-stubs/bin", "../../j2me-stubs/bin"}
	finalCP := cpPath
	for _, cand := range candidates {
		if info, err := os.Stat(cand); err == nil && info.IsDir() {
			if abs, err := filepath.Abs(cand); err == nil {
				finalCP = abs
				break
			}
		}
	}

	if _, err := os.Stat("/home/neuro/Downloads/KnightOnline_402.jar"); err == nil {
		finalCP = finalCP + ":/home/neuro/Downloads/KnightOnline_402.jar"
	}

	optString := "-Djava.class.path=" + finalCP

	cOpt := C.CString(optString)
	defer C.free(unsafe.Pointer(cOpt))

	var vm *C.JavaVM
	var env *C.JNIEnv

	res := C.create_java_vm(&vm, &env, cOpt)
	if res != C.JNI_OK {
		return nil, fmt.Errorf("JNI_CreateJavaVM failed with error code: %d", int(res))
	}

	globalJVM = &JVM{
		vm:  vm,
		env: env,
	}

	_ = globalJVM.RegisterNatives()

	return globalJVM, nil
}

// GetJVM returns the globally initialized JVM instance, or nil if not initialized.
func GetJVM() *JVM {
	jvmMutex.Lock()
	defer jvmMutex.Unlock()
	return globalJVM
}

// Destroy tears down the Java Virtual Machine.
func (j *JVM) Destroy() {
	jvmMutex.Lock()
	defer jvmMutex.Unlock()

	if j != nil && j.vm != nil {
		C.destroy_java_vm(j.vm)
		j.vm = nil
		j.env = nil
		globalJVM = nil
	}
}

// GetEnv returns the unsafe.Pointer representation of the JNIEnv.
func (j *JVM) GetEnv() unsafe.Pointer {
	if j == nil {
		return nil
	}
	return unsafe.Pointer(j.env)
}

// GetVM returns the unsafe.Pointer representation of the JavaVM.
func (j *JVM) GetVM() unsafe.Pointer {
	if j == nil {
		return nil
	}
	return unsafe.Pointer(j.vm)
}

// Env returns the native *C.JNIEnv pointer for internal CGO calls.
func (j *JVM) Env() *C.JNIEnv {
	if j == nil {
		return nil
	}
	return j.env
}

// VM returns the native *C.JavaVM pointer for internal CGO calls.
func (j *JVM) VM() *C.JavaVM {
	if j == nil {
		return nil
	}
	return j.vm
}
