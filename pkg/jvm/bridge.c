#include <jni.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "_cgo_export.h"

// --- Graphics Native Methods ---


static void bind_graphics_target(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "nativeTargetId", "I");
    if (fid != NULL) {
        jint targetId = (*env)->GetIntField(env, obj, fid);
        go_setActiveTarget(targetId);
    }
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_fillRect(JNIEnv *env, jobject obj, jint x, jint y, jint width, jint height) {
    bind_graphics_target(env, obj);
    go_fillRect(x, y, width, height);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawRect(JNIEnv *env, jobject obj, jint x, jint y, jint width, jint height) {
    bind_graphics_target(env, obj);
    go_drawRect(x, y, width, height);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawLine(JNIEnv *env, jobject obj, jint x1, jint y1, jint x2, jint y2) {
    bind_graphics_target(env, obj);
    go_drawLine(x1, y1, x2, y2);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_setColor1(JNIEnv *env, jobject obj, jint rgb) {
    bind_graphics_target(env, obj);
    go_setColor(rgb);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_setColor3(JNIEnv *env, jobject obj, jint r, jint g, jint b) {
    bind_graphics_target(env, obj);
    go_setColor3(r, g, b);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawString(JNIEnv *env, jobject obj, jstring str, jint x, jint y, jint anchor) {
    bind_graphics_target(env, obj);
    if (str == NULL) return;
    const char *cStr = (*env)->GetStringUTFChars(env, str, NULL);
    go_drawString((char*)cStr, x, y, anchor);
    (*env)->ReleaseStringUTFChars(env, str, cStr);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawSubstring(JNIEnv *env, jobject obj, jstring str, jint offset, jint len, jint x, jint y, jint anchor) {
    bind_graphics_target(env, obj);
    if (str == NULL) return;
    const char *cStr = (*env)->GetStringUTFChars(env, str, NULL);
    go_drawSubstring((char*)cStr, offset, len, x, y, anchor);
    (*env)->ReleaseStringUTFChars(env, str, cStr);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawChar(JNIEnv *env, jobject obj, jchar ch, jint x, jint y, jint anchor) {
    bind_graphics_target(env, obj);
    go_drawChar(ch, x, y, anchor);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawChars(JNIEnv *env, jobject obj, jcharArray chars, jint offset, jint len, jint x, jint y, jint anchor) {
    bind_graphics_target(env, obj);
    if (chars == NULL) return;
    jchar *cArr = (*env)->GetCharArrayElements(env, chars, NULL);
    go_drawChars(cArr, offset, len, x, y, anchor);
    (*env)->ReleaseCharArrayElements(env, chars, cArr, JNI_ABORT);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawImage(JNIEnv *env, jobject obj, jobject img, jint x, jint y, jint anchor) {
    bind_graphics_target(env, obj);
    if (img == NULL) return;
    jclass imgClass = (*env)->GetObjectClass(env, img);
    jfieldID fid = (*env)->GetFieldID(env, imgClass, "nativeId", "I");
    if (fid == NULL) return;
    jint imgId = (*env)->GetIntField(env, img, fid);
    go_drawImage(imgId, x, y, anchor);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawRegion(JNIEnv *env, jobject obj, jobject src, jint x_src, jint y_src, jint width, jint height, jint transform, jint x_dest, jint y_dest, jint anchor) {
    bind_graphics_target(env, obj);
    if (src == NULL) return;
    jclass imgClass = (*env)->GetObjectClass(env, src);
    jfieldID fid = (*env)->GetFieldID(env, imgClass, "nativeId", "I");
    if (fid == NULL) return;
    jint imgId = (*env)->GetIntField(env, src, fid);
    go_drawRegion(imgId, x_src, y_src, width, height, transform, x_dest, y_dest, anchor);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawRGB(JNIEnv *env, jobject obj, jintArray rgbData, jint offset, jint scanlength, jint x, jint y, jint width, jint height, jboolean processAlpha) {
    bind_graphics_target(env, obj);
    if (rgbData == NULL) return;
    jsize len = (*env)->GetArrayLength(env, rgbData);
    jint *data = (*env)->GetIntArrayElements(env, rgbData, NULL);
    go_drawRGB(data, len, offset, scanlength, x, y, width, height, processAlpha);
    (*env)->ReleaseIntArrayElements(env, rgbData, data, JNI_ABORT);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_setClip(JNIEnv *env, jobject obj, jint x, jint y, jint width, jint height) {
    bind_graphics_target(env, obj);
    go_setClip(x, y, width, height);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_clipRect(JNIEnv *env, jobject obj, jint x, jint y, jint width, jint height) {
    bind_graphics_target(env, obj);
    go_clipRect(x, y, width, height);
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Graphics_getClipX(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    return go_getClipX();
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Graphics_getClipY(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    return go_getClipY();
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Graphics_getClipWidth(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    return go_getClipWidth();
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Graphics_getClipHeight(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    return go_getClipHeight();
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_translate(JNIEnv *env, jobject obj, jint x, jint y) {
    bind_graphics_target(env, obj);
    go_translate(x, y);
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Graphics_getTranslateX(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    return go_getTranslateX();
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Graphics_getTranslateY(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    return go_getTranslateY();
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_fillArc(JNIEnv *env, jobject obj, jint x, jint y, jint w, jint h, jint start, jint arc) {
    bind_graphics_target(env, obj);
    go_fillArc(x, y, w, h, start, arc);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawArc(JNIEnv *env, jobject obj, jint x, jint y, jint w, jint h, jint start, jint arc) {
    bind_graphics_target(env, obj);
    go_drawArc(x, y, w, h, start, arc);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_fillRoundRect(JNIEnv *env, jobject obj, jint x, jint y, jint w, jint h, jint arcW, jint arcH) {
    bind_graphics_target(env, obj);
    go_fillRoundRect(x, y, w, h, arcW, arcH);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_drawRoundRect(JNIEnv *env, jobject obj, jint x, jint y, jint w, jint h, jint arcW, jint arcH) {
    bind_graphics_target(env, obj);
    go_drawRoundRect(x, y, w, h, arcW, arcH);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_fillTriangle(JNIEnv *env, jobject obj, jint x1, jint y1, jint x2, jint y2, jint x3, jint y3) {
    bind_graphics_target(env, obj);
    go_fillTriangle(x1, y1, x2, y2, x3, y3);
}

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Graphics_getFont(JNIEnv *env, jobject obj) {
    bind_graphics_target(env, obj);
    go_getFont();
    jclass fontClass = (*env)->FindClass(env, "javax/microedition/lcdui/Font");
    if (fontClass == NULL) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, fontClass, "<init>", "()V");
    if (ctor == NULL) return NULL;
    return (*env)->NewObject(env, fontClass, ctor);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_setFont(JNIEnv *env, jobject obj, jobject font) {
    bind_graphics_target(env, obj);
    go_setFont();
}


// --- Image Native Methods ---

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Image_createImageString(JNIEnv *env, jclass clazz, jstring path) {
    if (path == NULL) return NULL;
    const char *cPath = (*env)->GetStringUTFChars(env, path, NULL);
    jint id = go_createImageString((char*)cPath);
    (*env)->ReleaseStringUTFChars(env, path, cPath);
    if (id <= 0) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, clazz, "<init>", "(I)V");
    return (*env)->NewObject(env, clazz, ctor, id);
}

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Image_createImageWH(JNIEnv *env, jclass clazz, jint w, jint h) {
    jint id = go_createImageWH(w, h);
    if (id <= 0) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, clazz, "<init>", "(I)V");
    return (*env)->NewObject(env, clazz, ctor, id);
}

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Image_createImageBytes(JNIEnv *env, jclass clazz, jbyteArray data, jint offset, jint len) {
    if (data == NULL) return NULL;
    jbyte *bytes = (*env)->GetByteArrayElements(env, data, NULL);
    jint id = go_createImageBytes((char*)bytes, offset, len);
    (*env)->ReleaseByteArrayElements(env, data, bytes, JNI_ABORT);
    if (id <= 0) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, clazz, "<init>", "(I)V");
    return (*env)->NewObject(env, clazz, ctor, id);
}

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Image_createImageRegion(JNIEnv *env, jclass clazz, jobject src, jint x, jint y, jint w, jint h, jint transform) {
    if (src == NULL) return NULL;
    jclass srcClass = (*env)->GetObjectClass(env, src);
    jfieldID fid = (*env)->GetFieldID(env, srcClass, "nativeId", "I");
    if (fid == NULL) return NULL;
    jint srcId = (*env)->GetIntField(env, src, fid);
    jint id = go_createImageRegion(srcId, x, y, w, h, transform);
    if (id <= 0) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, clazz, "<init>", "(I)V");
    return (*env)->NewObject(env, clazz, ctor, id);
}

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Image_createRGBImage(JNIEnv *env, jclass clazz, jintArray rgb, jint width, jint height, jboolean processAlpha) {
    if (rgb == NULL) return NULL;
    jsize len = (*env)->GetArrayLength(env, rgb);
    jint *data = (*env)->GetIntArrayElements(env, rgb, NULL);
    jint id = go_createRGBImage(data, len, width, height, processAlpha);
    (*env)->ReleaseIntArrayElements(env, rgb, data, JNI_ABORT);
    if (id <= 0) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, clazz, "<init>", "(I)V");
    return (*env)->NewObject(env, clazz, ctor, id);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Image_getRGB(JNIEnv *env, jobject obj, jintArray rgbData, jint offset, jint scanlength, jint x, jint y, jint width, jint height) {
    if (rgbData == NULL) return;
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "nativeId", "I");
    if (fid == NULL) return;
    jint imgId = (*env)->GetIntField(env, obj, fid);

    jsize len = (*env)->GetArrayLength(env, rgbData);
    jint *data = (*env)->GetIntArrayElements(env, rgbData, NULL);
    go_getRGB(imgId, data, len, offset, scanlength, x, y, width, height);
    (*env)->ReleaseIntArrayElements(env, rgbData, data, 0);
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Image_getWidth(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "nativeId", "I");
    if (fid == NULL) return 0;
    jint id = (*env)->GetIntField(env, obj, fid);
    return go_imageGetWidth(id);
}

JNIEXPORT jint JNICALL Java_javax_microedition_lcdui_Image_getHeight(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "nativeId", "I");
    if (fid == NULL) return 0;
    jint id = (*env)->GetIntField(env, obj, fid);
    return go_imageGetHeight(id);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Image_destroy(JNIEnv *env, jclass clazz, jint id) {
    go_destroyImage(id);
}

JNIEXPORT jobject JNICALL Java_javax_microedition_lcdui_Image_getGraphics(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "nativeId", "I");
    if (fid == NULL) return NULL;
    jint imgId = (*env)->GetIntField(env, obj, fid);

    jclass gClazz = (*env)->FindClass(env, "javax/microedition/lcdui/Graphics");
    if (gClazz == NULL) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, gClazz, "<init>", "(I)V");
    if (ctor == NULL) return NULL;
    return (*env)->NewObject(env, gClazz, ctor, imgId);
}

// --- Canvas Native Methods ---

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Canvas_serviceRepaints(JNIEnv *env, jobject obj) {
    go_serviceRepaints();
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Canvas_flushGraphics(JNIEnv *env, jobject obj) {
    go_flushGraphics();
}

// --- Display Native Methods ---

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Display_nSetCurrent(JNIEnv *env, jclass clazz, jobject d) {
    go_setCurrentDisplayable(env, d);
}

JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Display_vibrate(JNIEnv *env, jobject obj, jint duration) {
    go_vibrate(duration);
}

JNIEXPORT jboolean JNICALL Java_javax_microedition_lcdui_Display_flashBacklight(JNIEnv *env, jobject obj, jint duration) {
    return go_flashBacklight(duration);
}

// --- RecordStore Native Methods ---

JNIEXPORT jobject JNICALL Java_javax_microedition_rms_RecordStore_openRecordStore(JNIEnv *env, jclass clazz, jstring name, jboolean create) {
    if (name == NULL) return NULL;
    const char *cName = (*env)->GetStringUTFChars(env, name, NULL);
    jint id = go_openRecordStore((char*)cName, create);
    (*env)->ReleaseStringUTFChars(env, name, cName);
    if (id <= 0) return NULL;
    jmethodID ctor = (*env)->GetMethodID(env, clazz, "<init>", "(Ljava/lang/String;I)V");
    return (*env)->NewObject(env, clazz, ctor, name, id);
}

JNIEXPORT void JNICALL Java_javax_microedition_rms_RecordStore_closeRecordStore(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "recordStoreId", "I");
    if (fid == NULL) return;
    jint handle = (*env)->GetIntField(env, obj, fid);
    go_closeRecordStore(handle);
}

JNIEXPORT void JNICALL Java_javax_microedition_rms_RecordStore_deleteRecordStore(JNIEnv *env, jclass clazz, jstring name) {
    if (name == NULL) return;
    const char *cName = (*env)->GetStringUTFChars(env, name, NULL);
    go_deleteRecordStore((char*)cName);
    (*env)->ReleaseStringUTFChars(env, name, cName);
}

JNIEXPORT jobjectArray JNICALL Java_javax_microedition_rms_RecordStore_listRecordStores(JNIEnv *env, jclass clazz) {
    jint count = go_listRecordStoresCount();
    jclass stringClass = (*env)->FindClass(env, "java/lang/String");
    jobjectArray arr = (*env)->NewObjectArray(env, count, stringClass, NULL);
    for (int i = 0; i < count; i++) {
        char *str = go_listRecordStoreName(i);
        if (str != NULL) {
            jstring jstr = (*env)->NewStringUTF(env, str);
            (*env)->SetObjectArrayElement(env, arr, i, jstr);
            free(str);
        }
    }
    return arr;
}

JNIEXPORT jint JNICALL Java_javax_microedition_rms_RecordStore_getNumRecords(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "recordStoreId", "I");
    if (fid == NULL) return 0;
    jint handle = (*env)->GetIntField(env, obj, fid);
    return go_getNumRecords(handle);
}

JNIEXPORT jbyteArray JNICALL Java_javax_microedition_rms_RecordStore_getRecord(JNIEnv *env, jobject obj, jint recId) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "recordStoreId", "I");
    if (fid == NULL) return NULL;
    jint handle = (*env)->GetIntField(env, obj, fid);
    jint sz = go_getRecordSize(handle, recId);
    if (sz < 0) return NULL;
    jbyteArray arr = (*env)->NewByteArray(env, sz);
    if (sz > 0) {
        char *buf = (char*)malloc(sz);
        go_getRecordData(handle, recId, buf);
        (*env)->SetByteArrayRegion(env, arr, 0, sz, (jbyte*)buf);
        free(buf);
    }
    return arr;
}

JNIEXPORT jint JNICALL Java_javax_microedition_rms_RecordStore_addRecord(JNIEnv *env, jobject obj, jbyteArray data, jint offset, jint len) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "recordStoreId", "I");
    if (fid == NULL) return -1;
    jint handle = (*env)->GetIntField(env, obj, fid);
    jbyte *bytes = NULL;
    if (data != NULL && len > 0) {
        bytes = (*env)->GetByteArrayElements(env, data, NULL);
    }
    jint recId = go_addRecord(handle, (char*)bytes, offset, len);
    if (bytes != NULL) {
        (*env)->ReleaseByteArrayElements(env, data, bytes, JNI_ABORT);
    }
    return recId;
}

JNIEXPORT void JNICALL Java_javax_microedition_rms_RecordStore_setRecord(JNIEnv *env, jobject obj, jint recId, jbyteArray data, jint offset, jint len) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "recordStoreId", "I");
    if (fid == NULL) return;
    jint handle = (*env)->GetIntField(env, obj, fid);
    jbyte *bytes = NULL;
    if (data != NULL && len > 0) {
        bytes = (*env)->GetByteArrayElements(env, data, NULL);
    }
    go_setRecord(handle, recId, (char*)bytes, offset, len);
    if (bytes != NULL) {
        (*env)->ReleaseByteArrayElements(env, data, bytes, JNI_ABORT);
    }
}

// --- RegisterNatives Implementation ---

static JNINativeMethod graphics_methods[] = {
    {"fillRect", "(IIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_fillRect},
    {"drawRect", "(IIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawRect},
    {"drawLine", "(IIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawLine},
    {"setColor", "(I)V", (void *)&Java_javax_microedition_lcdui_Graphics_setColor1},
    {"setColor", "(III)V", (void *)&Java_javax_microedition_lcdui_Graphics_setColor3},
    {"drawString", "(Ljava/lang/String;III)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawString},
    {"drawSubstring", "(Ljava/lang/String;IIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawSubstring},
    {"drawChar", "(CIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawChar},
    {"drawChars", "([CIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawChars},
    {"drawImage", "(Ljavax/microedition/lcdui/Image;III)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawImage},
    {"drawRegion", "(Ljavax/microedition/lcdui/Image;IIIIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawRegion},
    {"setClip", "(IIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_setClip},
    {"clipRect", "(IIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_clipRect},
    {"getClipX", "()I", (void *)&Java_javax_microedition_lcdui_Graphics_getClipX},
    {"getClipY", "()I", (void *)&Java_javax_microedition_lcdui_Graphics_getClipY},
    {"getClipWidth", "()I", (void *)&Java_javax_microedition_lcdui_Graphics_getClipWidth},
    {"getClipHeight", "()I", (void *)&Java_javax_microedition_lcdui_Graphics_getClipHeight},
    {"translate", "(II)V", (void *)&Java_javax_microedition_lcdui_Graphics_translate},
    {"getTranslateX", "()I", (void *)&Java_javax_microedition_lcdui_Graphics_getTranslateX},
    {"getTranslateY", "()I", (void *)&Java_javax_microedition_lcdui_Graphics_getTranslateY},
    {"fillArc", "(IIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_fillArc},
    {"drawArc", "(IIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawArc},
    {"fillRoundRect", "(IIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_fillRoundRect},
    {"drawRoundRect", "(IIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_drawRoundRect},
    {"fillTriangle", "(IIIIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_fillTriangle},
    {"getFont", "()Ljavax/microedition/lcdui/Font;", (void *)&Java_javax_microedition_lcdui_Graphics_getFont},
    {"setFont", "(Ljavax/microedition/lcdui/Font;)V", (void *)&Java_javax_microedition_lcdui_Graphics_setFont}
};

static JNINativeMethod image_methods[] = {
    {"createImage", "(II)Ljavax/microedition/lcdui/Image;", (void *)&Java_javax_microedition_lcdui_Image_createImageWH},
    {"createImage", "([BII)Ljavax/microedition/lcdui/Image;", (void *)&Java_javax_microedition_lcdui_Image_createImageBytes},
    {"createImage", "(Ljavax/microedition/lcdui/Image;IIIII)Ljavax/microedition/lcdui/Image;", (void *)&Java_javax_microedition_lcdui_Image_createImageRegion},
    {"getWidth", "()I", (void *)&Java_javax_microedition_lcdui_Image_getWidth},
    {"getHeight", "()I", (void *)&Java_javax_microedition_lcdui_Image_getHeight},
    {"getGraphics", "()Ljavax/microedition/lcdui/Graphics;", (void *)&Java_javax_microedition_lcdui_Image_getGraphics}
};

static JNINativeMethod canvas_methods[] = {
    {"flushGraphics", "()V", (void *)&Java_javax_microedition_lcdui_Canvas_flushGraphics}
};

static JNINativeMethod display_methods[] = {
    {"nSetCurrent", "(Ljavax/microedition/lcdui/Displayable;)V", (void *)&Java_javax_microedition_lcdui_Display_nSetCurrent},
    {"vibrate", "(I)V", (void *)&Java_javax_microedition_lcdui_Display_vibrate},
    {"flashBacklight", "(I)Z", (void *)&Java_javax_microedition_lcdui_Display_flashBacklight}
};

static JNINativeMethod recordstore_methods[] = {
    {"openRecordStore", "(Ljava/lang/String;Z)Ljavax/microedition/rms/RecordStore;", (void *)&Java_javax_microedition_rms_RecordStore_openRecordStore},
    {"closeRecordStore", "()V", (void *)&Java_javax_microedition_rms_RecordStore_closeRecordStore},
    {"deleteRecordStore", "(Ljava/lang/String;)V", (void *)&Java_javax_microedition_rms_RecordStore_deleteRecordStore},
    {"listRecordStores", "()[Ljava/lang/String;", (void *)&Java_javax_microedition_rms_RecordStore_listRecordStores},
    {"getNumRecords", "()I", (void *)&Java_javax_microedition_rms_RecordStore_getNumRecords},
    {"getRecord", "(I)[B", (void *)&Java_javax_microedition_rms_RecordStore_getRecord},
    {"addRecord", "([BII)I", (void *)&Java_javax_microedition_rms_RecordStore_addRecord},
    {"setRecord", "(I[BII)V", (void *)&Java_javax_microedition_rms_RecordStore_setRecord}
};

jint register_graphics_natives(JavaVM *vm, JNIEnv *env) {
    if (vm == NULL || *vm == NULL) return -1;
    JNIEnv *current_env = env;
    jint getEnvRes = (*vm)->GetEnv(vm, (void**)&current_env, JNI_VERSION_1_8);
    if (getEnvRes == JNI_EDETACHED) {
        if ((*vm)->AttachCurrentThread(vm, (void**)&current_env, NULL) != JNI_OK) {
            return -1;
        }
    }
    if (current_env == NULL || *current_env == NULL) return -1;
    if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);

    // Register Graphics
    jclass gClazz = (*current_env)->FindClass(current_env, "javax/microedition/lcdui/Graphics");
    if (gClazz != NULL) {
        (*current_env)->RegisterNatives(current_env, gClazz, graphics_methods, sizeof(graphics_methods)/sizeof(graphics_methods[0]));
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
    }

    // Register Image
    jclass imgClazz = (*current_env)->FindClass(current_env, "javax/microedition/lcdui/Image");
    if (imgClazz != NULL) {
        (*current_env)->RegisterNatives(current_env, imgClazz, image_methods, sizeof(image_methods)/sizeof(image_methods[0]));
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
    }

    // Register Canvas
    jclass cClazz = (*current_env)->FindClass(current_env, "javax/microedition/lcdui/Canvas");
    if (cClazz != NULL) {
        (*current_env)->RegisterNatives(current_env, cClazz, canvas_methods, sizeof(canvas_methods)/sizeof(canvas_methods[0]));
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
    }

    // Register Display
    jclass dClazz = (*current_env)->FindClass(current_env, "javax/microedition/lcdui/Display");
    if (dClazz != NULL) {
        (*current_env)->RegisterNatives(current_env, dClazz, display_methods, sizeof(display_methods)/sizeof(display_methods[0]));
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
    }

    // Register RecordStore
    jclass rClazz = (*current_env)->FindClass(current_env, "javax/microedition/rms/RecordStore");
    if (rClazz != NULL) {
        (*current_env)->RegisterNatives(current_env, rClazz, recordstore_methods, sizeof(recordstore_methods)/sizeof(recordstore_methods[0]));
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
    }

    return 0;
}

void c_dispatch_pointer_event(JavaVM *vm, JNIEnv *env, jobject canvas, jint x, jint y, jint event_type) {
    if (vm == NULL || canvas == NULL) return;
    JNIEnv *current_env = env;
    jint getEnvRes = (*vm)->GetEnv(vm, (void**)&current_env, JNI_VERSION_1_8);
    if (getEnvRes == JNI_EDETACHED) {
        if ((*vm)->AttachCurrentThread(vm, (void**)&current_env, NULL) != JNI_OK) return;
    }
    if (current_env == NULL || *current_env == NULL) return;
    jclass clazz = (*current_env)->GetObjectClass(current_env, canvas);
    if (clazz == NULL) return;
    const char *methodName = "pointerPressed";
    if (event_type == 1) methodName = "pointerReleased";
    else if (event_type == 2) methodName = "pointerDragged";
    jmethodID mid = (*current_env)->GetMethodID(current_env, clazz, methodName, "(II)V");
    if (mid != NULL) {
        (*current_env)->CallVoidMethod(current_env, canvas, mid, x, y);
        if ((*current_env)->ExceptionCheck(current_env)) {
            (*current_env)->ExceptionClear(current_env);
        }
    }
}

jint safe_invoke_main_args(JavaVM *vm, JNIEnv *env, const char *class_name, const char *arg0) {
    if (vm == NULL || *vm == NULL) return -1;
    JNIEnv *current_env = env;
    jint getEnvRes = (*vm)->GetEnv(vm, (void**)&current_env, JNI_VERSION_1_8);
    if (getEnvRes == JNI_EDETACHED) {
        if ((*vm)->AttachCurrentThread(vm, (void**)&current_env, NULL) != JNI_OK) {
            return -6;
        }
    }
    if (current_env == NULL || *current_env == NULL) return -1;
    if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);

    jclass clazz = (*current_env)->FindClass(current_env, class_name);
    if (clazz == NULL) {
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
        return -1;
    }

    jmethodID mainMethod = (*current_env)->GetStaticMethodID(current_env, clazz, "main", "([Ljava/lang/String;)V");
    if (mainMethod == NULL) {
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
        return -2;
    }

    jclass stringClass = (*current_env)->FindClass(current_env, "java/lang/String");
    if (stringClass == NULL) {
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
        return -3;
    }

    int argc = (arg0 != NULL && strlen(arg0) > 0) ? 1 : 0;
    jobjectArray args = (*current_env)->NewObjectArray(current_env, argc, stringClass, NULL);
    if (args == NULL) {
        if ((*current_env)->ExceptionCheck(current_env)) (*current_env)->ExceptionClear(current_env);
        return -4;
    }

    if (argc > 0) {
        jstring jarg0 = (*current_env)->NewStringUTF(current_env, arg0);
        (*current_env)->SetObjectArrayElement(current_env, args, 0, jarg0);
    }

    (*current_env)->CallStaticVoidMethod(current_env, clazz, mainMethod, args);

    if ((*current_env)->ExceptionCheck(current_env)) {
        (*current_env)->ExceptionDescribe(current_env);
        (*current_env)->ExceptionClear(current_env);
        return -5;
    }

    return 0;
}

jobject c_new_global_ref(JNIEnv *env, jobject obj) {
    if (env == NULL || obj == NULL) return NULL;
    return (*env)->NewGlobalRef(env, obj);
}

void c_delete_global_ref(JNIEnv *env, jobject obj) {
    if (env == NULL || obj == NULL) return;
    (*env)->DeleteGlobalRef(env, obj);
}

void c_dispatch_key_event(JavaVM *vm, JNIEnv *env, jobject canvas, jint key_code, jboolean pressed) {
    if (vm == NULL || canvas == NULL) return;
    JNIEnv *current_env = env;
    jint getEnvRes = (*vm)->GetEnv(vm, (void**)&current_env, JNI_VERSION_1_8);
    if (getEnvRes == JNI_EDETACHED) {
        if ((*vm)->AttachCurrentThread(vm, (void**)&current_env, NULL) != JNI_OK) return;
    }
    if (current_env == NULL || *current_env == NULL) return;
    jclass clazz = (*current_env)->GetObjectClass(current_env, canvas);
    if (clazz == NULL) return;
    const char *methodName = pressed ? "keyPressed" : "keyReleased";
    jmethodID mid = (*current_env)->GetMethodID(current_env, clazz, methodName, "(I)V");
    if (mid != NULL) {
        (*current_env)->CallVoidMethod(current_env, canvas, mid, key_code);
        if ((*current_env)->ExceptionCheck(current_env)) {
            (*current_env)->ExceptionClear(current_env);
        }
    }
}

void c_dispatch_key_repeated(JavaVM *vm, JNIEnv *env, jobject canvas, jint key_code) {
    if (vm == NULL || canvas == NULL) return;
    JNIEnv *current_env = env;
    jint getEnvRes = (*vm)->GetEnv(vm, (void**)&current_env, JNI_VERSION_1_8);
    if (getEnvRes == JNI_EDETACHED) {
        if ((*vm)->AttachCurrentThread(vm, (void**)&current_env, NULL) != JNI_OK) return;
    }
    if (current_env == NULL || *current_env == NULL) return;
    jclass clazz = (*current_env)->GetObjectClass(current_env, canvas);
    if (clazz == NULL) return;
    jmethodID mid = (*current_env)->GetMethodID(current_env, clazz, "keyRepeated", "(I)V");
    if (mid != NULL) {
        (*current_env)->CallVoidMethod(current_env, canvas, mid, key_code);
        if ((*current_env)->ExceptionCheck(current_env)) {
            (*current_env)->ExceptionClear(current_env);
        }
    }
}
void c_print_class_name(JNIEnv *env, jobject obj) {
    if (obj == NULL) return;
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jclass classClass = (*env)->FindClass(env, "java/lang/Class");
    jmethodID getName = (*env)->GetMethodID(env, classClass, "getName", "()Ljava/lang/String;");
    jstring nameStr = (jstring)(*env)->CallObjectMethod(env, clazz, getName);
    const char *name = (*env)->GetStringUTFChars(env, nameStr, NULL);
    printf("go_setCurrentDisplayable: %s\n", name);
    (*env)->ReleaseStringUTFChars(env, nameStr, name);
}

int c_is_textbox(JNIEnv *env, jobject obj) {
    if (obj == NULL) return 0;
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jclass tbClass = (*env)->FindClass(env, "javax/microedition/lcdui/TextBox");
    if (tbClass == NULL) {
        (*env)->ExceptionClear(env);
        return 0;
    }
    return (*env)->IsAssignableFrom(env, clazz, tbClass);
}

void c_handle_textbox_safe(JavaVM *vm, jobject obj, const char *new_text) {
    if (vm == NULL || obj == NULL) return;
    JNIEnv *env = NULL;
    jint getEnvRes = (*vm)->GetEnv(vm, (void**)&env, JNI_VERSION_1_8);
    if (getEnvRes == JNI_EDETACHED) {
        if ((*vm)->AttachCurrentThread(vm, (void**)&env, NULL) != JNI_OK) return;
    }
    if (env == NULL || *env == NULL) return;
    
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jmethodID setString = (*env)->GetMethodID(env, clazz, "setString", "(Ljava/lang/String;)V");
    if (setString != NULL) {
        jstring jstr = (*env)->NewStringUTF(env, new_text);
        (*env)->CallVoidMethod(env, obj, setString, jstr);
        if ((*env)->ExceptionCheck(env)) {
            (*env)->ExceptionClear(env);
            printf("Error calling setString\n");
        } else {
            printf("setString called successfully with: %s\n", new_text);
        }
    } else {
        printf("setString method NOT FOUND\n");
    }
    
    jmethodID getOk = (*env)->GetMethodID(env, clazz, "getOkCommand", "()Ljavax/microedition/lcdui/Command;");
    if (getOk != NULL) {
        jobject okCmd = (*env)->CallObjectMethod(env, obj, getOk);
        if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
        
        if (okCmd != NULL) {
            jmethodID fireCmd = (*env)->GetMethodID(env, clazz, "fireCommandAction", "(Ljavax/microedition/lcdui/Command;)V");
            if (fireCmd != NULL) {
                (*env)->CallVoidMethod(env, obj, fireCmd, okCmd);
                if ((*env)->ExceptionCheck(env)) {
                    (*env)->ExceptionClear(env);
                    printf("Error calling fireCommandAction\n");
                } else {
                    printf("fireCommandAction called successfully!\n");
                }
            } else {
                printf("fireCommandAction method NOT FOUND\n");
            }
        } else {
            printf("getOkCommand returned NULL\n");
        }
    } else {
        printf("getOkCommand method NOT FOUND\n");
    }
}

jint safe_invoke_main(JavaVM *vm, JNIEnv *env, const char *class_name) {
    return safe_invoke_main_args(vm, env, class_name, NULL);
}
