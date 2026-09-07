import re

with open('pkg/jvm/bridge.c', 'r') as f:
    content = f.read()

# Add bind_graphics_target function
insert_idx = content.find('JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Graphics_fillRect')
if insert_idx != -1:
    func_str = """
static void bind_graphics_target(JNIEnv *env, jobject obj) {
    jclass clazz = (*env)->GetObjectClass(env, obj);
    jfieldID fid = (*env)->GetFieldID(env, clazz, "nativeTargetId", "I");
    if (fid != NULL) {
        jint targetId = (*env)->GetIntField(env, obj, fid);
        go_setActiveTarget(targetId);
    }
}

"""
    content = content[:insert_idx] + func_str + content[insert_idx:]

# Add bind_graphics_target(env, obj); to all Graphics methods
pattern = r'(JNIEXPORT [^\{]+\{)'

def replace_func(match):
    decl = match.group(1)
    if 'Java_javax_microedition_lcdui_Graphics_' in decl and 'bind_graphics_target' not in decl:
        return decl + '\n    bind_graphics_target(env, obj);'
    return decl

content = re.sub(pattern, replace_func, content)

with open('pkg/jvm/bridge.c', 'w') as f:
    f.write(content)
