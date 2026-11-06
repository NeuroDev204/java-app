//go:build cgo

package autoupgrade

/*
#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux
#cgo LDFLAGS: -L${SRCDIR}/../../lib -L/usr/lib/jvm/java-21-openjdk-amd64/lib/server -Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server -ljvm

#include <jni.h>
#include <stdlib.h>
#include <string.h>

static jint autoupgrade_get_env(JavaVM *vm, JNIEnv **env_out) {
    if (vm == NULL || *vm == NULL) return JNI_ERR;
    JNIEnv *env = NULL;
    jint res = (*vm)->GetEnv(vm, (void**)&env, JNI_VERSION_1_8);
    if (res == JNI_EDETACHED) {
        res = (*vm)->AttachCurrentThread(vm, (void**)&env, NULL);
        if (res != JNI_OK) {
            return res;
        }
    }
    *env_out = env;
    return JNI_OK;
}

static char* autoupgrade_get_inventory_json(JavaVM *vm) {
    if (vm == NULL) return NULL;
    JNIEnv *env = NULL;
    if (autoupgrade_get_env(vm, &env) != JNI_OK || env == NULL) return NULL;

    if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);

    jclass clazz = (*env)->FindClass(env, "com/emulator/AutoUpgrade");
    if (clazz == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        return NULL;
    }

    jmethodID mid = (*env)->GetStaticMethodID(env, clazz, "getInventoryJson", "()Ljava/lang/String;");
    if (mid == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return NULL;
    }

    jstring jstr = (jstring)(*env)->CallStaticObjectMethod(env, clazz, mid);
    if ((*env)->ExceptionCheck(env)) {
        (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return NULL;
    }

    char *result = NULL;
    if (jstr != NULL) {
        const char *utf = (*env)->GetStringUTFChars(env, jstr, NULL);
        if (utf != NULL) {
            result = strdup(utf);
            (*env)->ReleaseStringUTFChars(env, jstr, utf);
        }
        (*env)->DeleteLocalRef(env, jstr);
    }
    (*env)->DeleteLocalRef(env, clazz);
    return result;
}

static jint autoupgrade_start_upgrade(JavaVM *vm, jint itemId, jint itemType, jint targetLevel,
                                      jint clover3Min, jint clover3Max, jint clover4Min, jint clover4Max,
                                      jboolean stopOutOfClover, jint delayMs) {
    if (vm == NULL) return -1;
    JNIEnv *env = NULL;
    if (autoupgrade_get_env(vm, &env) != JNI_OK || env == NULL) return -2;

    if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);

    jclass clazz = (*env)->FindClass(env, "com/emulator/AutoUpgrade");
    if (clazz == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        return -3;
    }

    jmethodID mid = (*env)->GetStaticMethodID(env, clazz, "startUpgrade", "(IIIIIIIZI)V");
    if (mid == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return -4;
    }

    (*env)->CallStaticVoidMethod(env, clazz, mid, itemId, itemType, targetLevel,
                                 clover3Min, clover3Max, clover4Min, clover4Max,
                                 stopOutOfClover, delayMs);

    if ((*env)->ExceptionCheck(env)) {
        (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return -5;
    }

    (*env)->DeleteLocalRef(env, clazz);
    return 0;
}

static jint autoupgrade_stop_upgrade(JavaVM *vm) {
    if (vm == NULL) return -1;
    JNIEnv *env = NULL;
    if (autoupgrade_get_env(vm, &env) != JNI_OK || env == NULL) return -2;

    if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);

    jclass clazz = (*env)->FindClass(env, "com/emulator/AutoUpgrade");
    if (clazz == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        return -3;
    }

    jmethodID mid = (*env)->GetStaticMethodID(env, clazz, "stopUpgrade", "()V");
    if (mid == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return -4;
    }

    (*env)->CallStaticVoidMethod(env, clazz, mid);

    if ((*env)->ExceptionCheck(env)) {
        (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return -5;
    }

    (*env)->DeleteLocalRef(env, clazz);
    return 0;
}

static char* autoupgrade_get_upgrade_status_json(JavaVM *vm) {
    if (vm == NULL) return NULL;
    JNIEnv *env = NULL;
    if (autoupgrade_get_env(vm, &env) != JNI_OK || env == NULL) return NULL;

    if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);

    jclass clazz = (*env)->FindClass(env, "com/emulator/AutoUpgrade");
    if (clazz == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        return NULL;
    }

    jmethodID mid = (*env)->GetStaticMethodID(env, clazz, "getUpgradeStatusJson", "()Ljava/lang/String;");
    if (mid == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return NULL;
    }

    jstring jstr = (jstring)(*env)->CallStaticObjectMethod(env, clazz, mid);
    if ((*env)->ExceptionCheck(env)) {
        (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return NULL;
    }

    char *result = NULL;
    if (jstr != NULL) {
        const char *utf = (*env)->GetStringUTFChars(env, jstr, NULL);
        if (utf != NULL) {
            result = strdup(utf);
            (*env)->ReleaseStringUTFChars(env, jstr, utf);
        }
        (*env)->DeleteLocalRef(env, jstr);
    }
    (*env)->DeleteLocalRef(env, clazz);
    return result;
}

static jint autoupgrade_reset_clover(JavaVM *vm) {
    if (vm == NULL) return -1;
    JNIEnv *env = NULL;
    if (autoupgrade_get_env(vm, &env) != JNI_OK || env == NULL) return -2;

    if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);

    jclass clazz = (*env)->FindClass(env, "com/emulator/AutoUpgrade");
    if (clazz == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        return -3;
    }

    jmethodID mid = (*env)->GetStaticMethodID(env, clazz, "resetCloverSlot", "()V");
    if (mid == NULL) {
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return -4;
    }

    (*env)->CallStaticVoidMethod(env, clazz, mid);
    if ((*env)->ExceptionCheck(env)) {
        (*env)->ExceptionClear(env);
        (*env)->DeleteLocalRef(env, clazz);
        return -5;
    }
    (*env)->DeleteLocalRef(env, clazz);
    return 0;
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	"java-app/pkg/jvm"
)

// JNIBridge implements the Bridge interface via CGO JNI calls to com.emulator.AutoUpgrade.
type JNIBridge struct{}

func (b *JNIBridge) getVM() (*C.JavaVM, error) {
	globalVM := jvm.GetGlobalJVM()
	if globalVM == nil {
		globalVM = jvm.GetJVM()
	}
	if globalVM == nil {
		return nil, fmt.Errorf("HotSpot JVM is not initialized")
	}
	rawVM := globalVM.GetVM()
	if rawVM == nil {
		return nil, fmt.Errorf("JVM pointer is nil")
	}
	return (*C.JavaVM)(rawVM), nil
}

// GetInventoryJson invokes com.emulator.AutoUpgrade.getInventoryJson().
func (b *JNIBridge) GetInventoryJson() (string, error) {
	vm, err := b.getVM()
	if err != nil {
		return "", err
	}
	cStr := C.autoupgrade_get_inventory_json(vm)
	if cStr == nil {
		return "", fmt.Errorf("failed to retrieve inventory from com.emulator.AutoUpgrade")
	}
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr), nil
}

// StartUpgrade invokes com.emulator.AutoUpgrade.startUpgrade(...).
func (b *JNIBridge) StartUpgrade(itemId, itemType, targetLevel, clover3Min, clover3Max, clover4Min, clover4Max int, stopOutOfClover bool, delayMs int) error {
	vm, err := b.getVM()
	if err != nil {
		return err
	}
	var cStop C.jboolean = C.JNI_FALSE
	if stopOutOfClover {
		cStop = C.JNI_TRUE
	}
	res := C.autoupgrade_start_upgrade(
		vm,
		C.jint(itemId),
		C.jint(itemType),
		C.jint(targetLevel),
		C.jint(clover3Min),
		C.jint(clover3Max),
		C.jint(clover4Min),
		C.jint(clover4Max),
		cStop,
		C.jint(delayMs),
	)
	if res != 0 {
		return fmt.Errorf("failed to start upgrade on com.emulator.AutoUpgrade (code %d)", int(res))
	}
	return nil
}

// StopUpgrade invokes com.emulator.AutoUpgrade.stopUpgrade().
func (b *JNIBridge) StopUpgrade() error {
	vm, err := b.getVM()
	if err != nil {
		return err
	}
	res := C.autoupgrade_stop_upgrade(vm)
	if res != 0 {
		return fmt.Errorf("failed to stop upgrade on com.emulator.AutoUpgrade (code %d)", int(res))
	}
	return nil
}

// GetUpgradeStatusJson invokes com.emulator.AutoUpgrade.getUpgradeStatusJson().
func (b *JNIBridge) GetUpgradeStatusJson() (string, error) {
	vm, err := b.getVM()
	if err != nil {
		return "", err
	}
	cStr := C.autoupgrade_get_upgrade_status_json(vm)
	if cStr == nil {
		return "", fmt.Errorf("failed to retrieve upgrade status from com.emulator.AutoUpgrade")
	}
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr), nil
}

// ResetClover invokes com.emulator.AutoUpgrade.resetCloverSlot().
func (b *JNIBridge) ResetClover() error {
	vm, err := b.getVM()
	if err != nil {
		return err
	}
	res := C.autoupgrade_reset_clover(vm)
	if res != 0 {
		return fmt.Errorf("failed to reset clover on com.emulator.AutoUpgrade (code %d)", int(res))
	}
	return nil
}

func defaultBridge() Bridge {
	return &JNIBridge{}
}
