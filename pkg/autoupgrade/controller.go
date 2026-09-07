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
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	"java-app/pkg/jvm"
)

// Item represents an equipment piece in the player's bag.
type Item struct {
	ID       int16  `json:"id"`
	Name     string `json:"name"`
	Type     int8   `json:"type"`
	Level    int8   `json:"level"`
	Quantity int    `json:"quantity"`
}

// InventoryData models the parsed inventory payload from Java runtime.
type InventoryData struct {
	Gold           int     `json:"gold"`
	Clover3Count   int     `json:"clover3Count"`
	Clover4Count   int     `json:"clover4Count"`
	InsuranceCount int     `json:"insuranceCount"`
	Items          []*Item `json:"items"`
}

// UpgradeSettings holds configuration parameters for the automated upgrade process.
type UpgradeSettings struct {
	SelectedItemIndex int    `json:"selectedItemIndex"`
	TargetLevel       int8   `json:"targetLevel"`
	Clover3Min        int8   `json:"clover3Min"`
	Clover3Max        int8   `json:"clover3Max"`
	Clover4Min        int8   `json:"clover4Min"`
	Clover4Max        int8   `json:"clover4Max"`
	StopIfOutOfClover bool   `json:"stopIfOutOfClover"`
	DelayMs           int    `json:"delayMs"`
	ItemID            int16  `json:"itemId,omitempty"`
	ItemType          int8   `json:"itemType,omitempty"`
	ItemName          string `json:"itemName,omitempty"`
}

// Status represents the real-time state of the automated upgrade routine.
type Status struct {
	Running        bool   `json:"running"`
	ItemID         int16  `json:"itemId"`
	ItemName       string `json:"itemName"`
	ItemType       int8   `json:"itemType"`
	CurrentLevel   int8   `json:"currentLevel"`
	TargetLevel    int8   `json:"targetLevel"`
	Clover3Count   int    `json:"clover3Count"`
	Clover4Count   int    `json:"clover4Count"`
	InsuranceCount int    `json:"insuranceCount"`
	Attempts       int    `json:"attempts"`
	Successes      int    `json:"successes"`
	Failures       int    `json:"failures"`
	Message        string `json:"message"`
	Stopped        bool   `json:"stopped"`
}

// ControllerInterface defines the common methods for upgrade controllers.
type ControllerInterface interface {
	RefreshInventory() (*InventoryData, error)
	GetCachedInventory() *InventoryData
	GetCachedStatus() *Status
	GetSettings() *UpgradeSettings
	UpdateSettings(s *UpgradeSettings)
	IsRunning() bool
	StartUpgrade(settings *UpgradeSettings) error
	StopUpgrade() error
	GetStatus() (*Status, error)
}

// Bridge abstracts the low-level communication with the Java runtime.
type Bridge interface {
	GetInventoryJson() (string, error)
	StartUpgrade(itemId, itemType, targetLevel, clover3Min, clover3Max, clover4Min, clover4Max int, stopOutOfClover bool, delayMs int) error
	StopUpgrade() error
	GetUpgradeStatusJson() (string, error)
}

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

// DefaultSettings returns safe and sensible defaults for auto upgrading.
func DefaultSettings() *UpgradeSettings {
	return &UpgradeSettings{
		SelectedItemIndex: 0,
		TargetLevel:       8,
		Clover3Min:        4,
		Clover3Max:        7,
		Clover4Min:        8,
		Clover4Max:        15,
		StopIfOutOfClover: true,
		DelayMs:           500,
	}
}

// Controller coordinates the auto-upgrade logic between Go UI and Java runtime.
type Controller struct {
	bridge        Bridge
	mu            sync.RWMutex
	settings      *UpgradeSettings
	lastInventory *InventoryData
	lastStatus    *Status
}

// NewController initializes a new Controller with the standard JNIBridge.
func NewController() *Controller {
	return NewControllerWithBridge(&JNIBridge{})
}

// NewControllerWithBridge initializes a Controller with a custom Bridge implementation.
func NewControllerWithBridge(b Bridge) *Controller {
	if b == nil {
		b = &JNIBridge{}
	}
	return &Controller{
		bridge:   b,
		settings: DefaultSettings(),
	}
}

// SetBridge allows swapping the underlying Bridge implementation.
func (c *Controller) SetBridge(b Bridge) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if b == nil {
		b = &JNIBridge{}
	}
	c.bridge = b
}

// GetBridge returns the current active Bridge implementation.
func (c *Controller) GetBridge() Bridge {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bridge
}

// RefreshInventory fetches the latest bag contents and counts from the Java session.
func (c *Controller) RefreshInventory() (*InventoryData, error) {
	c.mu.RLock()
	bridge := c.bridge
	c.mu.RUnlock()

	if bridge == nil {
		return nil, fmt.Errorf("bridge is nil")
	}

	jsonStr, err := bridge.GetInventoryJson()
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	var data InventoryData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse inventory JSON: %w (raw: %s)", err, jsonStr)
	}

	c.mu.Lock()
	c.lastInventory = &data
	c.mu.Unlock()

	return &data, nil
}

// GetCachedInventory returns the most recently fetched inventory snapshot.
func (c *Controller) GetCachedInventory() *InventoryData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastInventory == nil {
		return nil
	}
	invCopy := *c.lastInventory
	return &invCopy
}

// GetSettings returns a copy of current upgrade configuration settings.
func (c *Controller) GetSettings() *UpgradeSettings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.settings == nil {
		return DefaultSettings()
	}
	sCopy := *c.settings
	return &sCopy
}

// UpdateSettings updates the configuration settings.
func (c *Controller) UpdateSettings(s *UpgradeSettings) {
	if s == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	sCopy := *s
	c.settings = &sCopy
}

// StartUpgrade initiates the auto-upgrade routine for the chosen equipment.
func (c *Controller) StartUpgrade(settings *UpgradeSettings) error {
	if settings == nil {
		settings = c.GetSettings()
	}

	if settings.TargetLevel < 1 || settings.TargetLevel > 15 {
		return fmt.Errorf("target level must be between 1 and 15 (got %d)", settings.TargetLevel)
	}

	c.mu.Lock()
	bridge := c.bridge
	inv := c.lastInventory
	c.mu.Unlock()

	if bridge == nil {
		return fmt.Errorf("bridge is nil")
	}

	// Resolve item if ItemID is 0 or if SelectedItemIndex is specified
	if settings.ItemID == 0 {
		if inv == nil || len(inv.Items) == 0 {
			return fmt.Errorf("no equipment found in inventory")
		}
		if settings.SelectedItemIndex < 0 || settings.SelectedItemIndex >= len(inv.Items) {
			return fmt.Errorf("selected item index %d out of bounds (len=%d)", settings.SelectedItemIndex, len(inv.Items))
		}
		item := inv.Items[settings.SelectedItemIndex]
		if item == nil {
			return fmt.Errorf("item at index %d is nil", settings.SelectedItemIndex)
		}
		settings.ItemID = item.ID
		settings.ItemType = item.Type
		settings.ItemName = item.Name
	}

	// Use JNI bridge to start upgrade
	err := bridge.StartUpgrade(
		int(settings.ItemID),
		int(settings.ItemType),
		int(settings.TargetLevel),
		int(settings.Clover3Min),
		int(settings.Clover3Max),
		int(settings.Clover4Min),
		int(settings.Clover4Max),
		settings.StopIfOutOfClover,
		settings.DelayMs,
	)
	if err != nil {
		return fmt.Errorf("failed to start upgrade: %w", err)
	}

	c.UpdateSettings(settings)
	c.mu.Lock()
	if c.lastStatus == nil {
		c.lastStatus = &Status{}
	}
	c.lastStatus.Running = true
	c.lastStatus.Stopped = false
	c.lastStatus.ItemID = settings.ItemID
	c.lastStatus.ItemType = settings.ItemType
	c.lastStatus.ItemName = settings.ItemName
	c.lastStatus.TargetLevel = settings.TargetLevel
	c.lastStatus.Message = fmt.Sprintf("Bat dau dap %s -> +%d", settings.ItemName, settings.TargetLevel)
	c.mu.Unlock()

	return nil
}

// StopUpgrade requests the auto-upgrade routine to halt immediately.
func (c *Controller) StopUpgrade() error {
	c.mu.RLock()
	bridge := c.bridge
	c.mu.RUnlock()

	if bridge == nil {
		return fmt.Errorf("bridge is nil")
	}

	if err := bridge.StopUpgrade(); err != nil {
		return fmt.Errorf("failed to stop upgrade: %w", err)
	}

	c.mu.Lock()
	if c.lastStatus != nil {
		c.lastStatus.Running = false
		c.lastStatus.Stopped = true
	}
	c.mu.Unlock()
	return nil
}

// GetStatus queries and parses the latest upgrade progress and metrics.
func (c *Controller) GetStatus() (*Status, error) {
	c.mu.RLock()
	bridge := c.bridge
	c.mu.RUnlock()

	if bridge == nil {
		return nil, fmt.Errorf("bridge is nil")
	}

	jsonStr, err := bridge.GetUpgradeStatusJson()
	if err != nil {
		return nil, fmt.Errorf("failed to get upgrade status: %w", err)
	}

	var status Status
	if err := json.Unmarshal([]byte(jsonStr), &status); err != nil {
		return nil, fmt.Errorf("failed to parse status JSON: %w (raw: %s)", err, jsonStr)
	}

	c.mu.Lock()
	c.lastStatus = &status
	c.mu.Unlock()

	return &status, nil
}

// GetCachedStatus returns the last retrieved Status without making a new bridge call.
func (c *Controller) GetCachedStatus() *Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil {
		return nil
	}
	sCopy := *c.lastStatus
	return &sCopy
}

// IsRunning indicates if the auto-upgrade routine is currently active.
func (c *Controller) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus != nil {
		return c.lastStatus.Running
	}
	return false
}
