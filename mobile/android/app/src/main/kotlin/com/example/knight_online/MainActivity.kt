package com.example.knight_online

import android.content.Context
import android.os.Handler
import android.os.Looper
import android.util.Log
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.util.concurrent.Executors

/**
 * MainActivity wires the Flutter MethodChannel
 * "com.example.knight_online/game_bridge" to the native Go bridge.
 *
 * The Go gamebridge package is loaded from a gomobile AAR when present.
 * Reflection keeps APKs compiling until that AAR is on the classpath;
 * loadJAR then returns "native bridge not linked" instead of faking a
 * checkerboard framebuffer.
 */
class MainActivity : FlutterActivity() {

    companion object {
        private const val CHANNEL = "com.example.knight_online/game_bridge"
        private const val TAG = "GameBridge"
        private const val NOT_LINKED = "native bridge not linked"
        private const val EMPTY_UPGRADE_STATUS =
            """{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}"""
        private val BRIDGE_CLASS_NAMES = arrayOf(
            "gamebridge.Bridge",
            "go.gamebridge.Bridge",
            "go.Bridge",
        )
    }

    // Background executor — keeps heavy VM calls off the main thread.
    private val executor = Executors.newSingleThreadExecutor()
    private val mainHandler = Handler(Looper.getMainLooper())

    // gomobile Bridge instance, or null when the AAR is not on the classpath.
    private var nativeBridge: Any? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        nativeBridge = tryLinkNativeBridge()

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL)
            .setMethodCallHandler { call: MethodCall, result: MethodChannel.Result ->
                handleBridgeCall(call, result)
            }
    }

    private fun handleBridgeCall(call: MethodCall, result: MethodChannel.Result) {
        when (call.method) {
            "loadJAR" -> {
                val jarBytes = call.arguments as? ByteArray
                if (jarBytes == null) {
                    result.error("INVALID_ARG", "loadJAR requires ByteArray", null)
                    return
                }
                // Run VM boot on background thread — can take 100–500 ms.
                executor.submit {
                    val error = nativeLoadJAR(jarBytes)
                    mainHandler.post {
                        result.success(error)
                    }
                }
            }

            "getFrameBuffer" -> {
                executor.submit {
                    val buf = nativeGetFrameBuffer()
                    mainHandler.post {
                        result.success(buf)
                    }
                }
            }

            "sendTouchDown" -> {
                val x = call.argument<Int>("x") ?: 0
                val y = call.argument<Int>("y") ?: 0
                executor.submit { nativeSendTouchDown(x, y) }
                result.success(null)
            }

            "sendTouchUp" -> {
                val x = call.argument<Int>("x") ?: 0
                val y = call.argument<Int>("y") ?: 0
                executor.submit { nativeSendTouchUp(x, y) }
                result.success(null)
            }

            "sendTouchDrag" -> {
                val x = call.argument<Int>("x") ?: 0
                val y = call.argument<Int>("y") ?: 0
                executor.submit { nativeSendTouchDrag(x, y) }
                result.success(null)
            }

            "sendKeyPress" -> {
                val keyCode = call.argument<Int>("keyCode") ?: -5
                executor.submit { nativeSendKeyPress(keyCode) }
                result.success(null)
            }

            "setShouldPaint" -> {
                val paint = call.arguments as? Boolean ?: true
                executor.submit { nativeSetShouldPaint(paint) }
                result.success(null)
            }

            "startUpgrade" -> {
                val configJSON = call.arguments as? String ?: ""
                executor.submit {
                    val error = nativeStartUpgrade(configJSON)
                    mainHandler.post {
                        result.success(error)
                    }
                }
            }

            "stopUpgrade" -> {
                executor.submit { nativeStopUpgrade() }
                result.success(null)
            }

            "getUpgradeStatus" -> {
                executor.submit {
                    val status = nativeGetUpgradeStatus()
                    mainHandler.post {
                        result.success(status)
                    }
                }
            }

            "stop" -> {
                executor.submit { nativeStop() }
                result.success(null)
            }

            "isStarted" -> {
                result.success(nativeIsStarted())
            }

            "getTextInputInfo" -> {
                executor.submit {
                    val info = nativeGetTextInputInfo()
                    mainHandler.post {
                        result.success(info)
                    }
                }
            }

            "submitTextInput" -> {
                val text = call.arguments as? String ?: ""
                executor.submit { nativeSubmitTextInput(text) }
                result.success(null)
            }

            "cancelTextInput" -> {
                executor.submit { nativeCancelTextInput() }
                result.success(null)
            }

            else -> result.notImplemented()
        }
    }

    private fun tryLinkNativeBridge(): Any? {
        try {
            Class.forName("go.Seq")
                .getMethod("setContext", Context::class.java)
                .invoke(null, applicationContext)
        } catch (e: ClassNotFoundException) {
            Log.i(TAG, "go.Seq not on classpath (AAR not linked yet)")
        } catch (e: Throwable) {
            Log.e(TAG, "go.Seq.setContext failed", e)
        }

        val clazz = loadBridgeClass()
        if (clazz == null) {
            Log.w(TAG, "gamebridge Bridge class missing; $NOT_LINKED")
            return null
        }
        return try {
            clazz.getDeclaredConstructor().newInstance()
        } catch (e: Throwable) {
            Log.e(TAG, "failed to instantiate gamebridge Bridge", e)
            null
        }
    }

    private fun loadBridgeClass(): Class<*>? {
        for (name in BRIDGE_CLASS_NAMES) {
            try {
                return Class.forName(name)
            } catch (e: ClassNotFoundException) {
                Log.i(TAG, "$name not found")
            }
        }
        return null
    }

    private fun nativeLoadJAR(jarBytes: ByteArray): String {
        val bridge = nativeBridge ?: return NOT_LINKED
        return try {
            (invokeNative(bridge, "loadJAR", arrayOf(ByteArray::class.java), jarBytes) as? String) ?: ""
        } catch (e: Throwable) {
            Log.e(TAG, "loadJAR failed", e)
            "native loadJAR failed: ${e.message}"
        }
    }

    private fun nativeGetFrameBuffer(): ByteArray? {
        val bridge = nativeBridge ?: return null
        return try {
            val raw = try {
                invokeNative(bridge, "getFrameBufferIfChanged", emptyArray())
            } catch (_: Throwable) {
                invokeNative(bridge, "getFrameBuffer", emptyArray())
            }
            when (raw) {
                is ByteArray -> raw
                null -> null
                else -> {
                    Log.e(TAG, "getFrameBuffer unexpected type ${raw.javaClass.name}")
                    null
                }
            }
        } catch (e: Throwable) {
            Log.e(TAG, "getFrameBuffer failed", e)
            null
        }
    }

    private fun nativeSendTouchDown(x: Int, y: Int) {
        invokeTouch("sendTouchDown", x, y)
    }

    private fun nativeSendTouchUp(x: Int, y: Int) {
        invokeTouch("sendTouchUp", x, y)
    }

    private fun nativeSendTouchDrag(x: Int, y: Int) {
        invokeTouch("sendTouchDrag", x, y)
    }

    private fun nativeSendKeyPress(keyCode: Int) {
        val bridge = nativeBridge ?: return
        try {
            // gomobile maps Go int to Java long.
            invokeNative(
                bridge,
                "sendKeyPress",
                arrayOf(java.lang.Long.TYPE),
                keyCode.toLong(),
            )
        } catch (e: NoSuchMethodException) {
            try {
                invokeNative(
                    bridge,
                    "sendKeyPress",
                    arrayOf(java.lang.Integer.TYPE),
                    keyCode,
                )
            } catch (inner: Throwable) {
                Log.e(TAG, "sendKeyPress failed", inner)
            }
        } catch (e: Throwable) {
            Log.e(TAG, "sendKeyPress failed", e)
        }
    }

    private fun nativeGetTextInputInfo(): String {
        val bridge = nativeBridge ?: return """{"active":false}"""
        return try {
            (invokeNative(bridge, "getTextInputInfoJSON", emptyArray()) as? String)
                ?: """{"active":false}"""
        } catch (e: Throwable) {
            Log.e(TAG, "getTextInputInfoJSON failed", e)
            """{"active":false}"""
        }
    }

    private fun nativeSubmitTextInput(text: String) {
        val bridge = nativeBridge ?: return
        try {
            invokeNative(
                bridge,
                "submitTextInput",
                arrayOf(String::class.java),
                text
            )
        } catch (e: Throwable) {
            Log.e(TAG, "submitTextInput failed", e)
        }
    }

    private fun nativeCancelTextInput() {
        val bridge = nativeBridge ?: return
        try {
            invokeNative(bridge, "cancelTextInput", emptyArray())
        } catch (e: Throwable) {
            Log.e(TAG, "cancelTextInput failed", e)
        }
    }

    private fun nativeSetShouldPaint(paint: Boolean) {
        val bridge = nativeBridge ?: return
        try {
            invokeNative(
                bridge,
                "setShouldPaint",
                arrayOf(java.lang.Boolean.TYPE),
                paint,
            )
        } catch (e: Throwable) {
            Log.e(TAG, "setShouldPaint failed", e)
        }
    }

    private fun nativeStartUpgrade(configJSON: String): String {
        val bridge = nativeBridge ?: return NOT_LINKED
        return try {
            (invokeNative(bridge, "startUpgrade", arrayOf(String::class.java), configJSON) as? String)
                ?: ""
        } catch (e: Throwable) {
            Log.e(TAG, "startUpgrade failed", e)
            "native startUpgrade failed: ${e.message}"
        }
    }

    private fun nativeStopUpgrade() {
        val bridge = nativeBridge ?: return
        try {
            invokeNative(bridge, "stopUpgrade", emptyArray())
        } catch (e: Throwable) {
            Log.e(TAG, "stopUpgrade failed", e)
        }
    }

    private fun nativeGetUpgradeStatus(): String {
        val bridge = nativeBridge ?: return EMPTY_UPGRADE_STATUS
        return try {
            (invokeNative(bridge, "getUpgradeStatus", emptyArray()) as? String) ?: EMPTY_UPGRADE_STATUS
        } catch (e: Throwable) {
            Log.e(TAG, "getUpgradeStatus failed", e)
            ""
        }
    }

    private fun nativeStop() {
        val bridge = nativeBridge ?: return
        try {
            invokeNative(bridge, "stop", emptyArray())
        } catch (e: Throwable) {
            Log.e(TAG, "stop failed", e)
        }
    }

    private fun nativeIsStarted(): Boolean {
        val bridge = nativeBridge ?: return false
        return try {
            invokeNative(bridge, "isStarted", emptyArray()) as? Boolean ?: false
        } catch (e: Throwable) {
            Log.e(TAG, "isStarted failed", e)
            false
        }
    }

    private fun invokeTouch(method: String, x: Int, y: Int) {
        val bridge = nativeBridge ?: return
        try {
            // gomobile maps Go int to Java long.
            invokeNative(
                bridge,
                method,
                arrayOf(java.lang.Long.TYPE, java.lang.Long.TYPE),
                x.toLong(),
                y.toLong(),
            )
        } catch (e: NoSuchMethodException) {
            try {
                invokeNative(
                    bridge,
                    method,
                    arrayOf(java.lang.Integer.TYPE, java.lang.Integer.TYPE),
                    x,
                    y,
                )
            } catch (inner: Throwable) {
                Log.e(TAG, "$method failed", inner)
            }
        } catch (e: Throwable) {
            Log.e(TAG, "$method failed", e)
        }
    }

    private fun invokeNative(
        bridge: Any,
        name: String,
        paramTypes: Array<Class<*>>,
        vararg args: Any?,
    ): Any? {
        val method = bridge.javaClass.getMethod(name, *paramTypes)
        return method.invoke(bridge, *args)
    }

    override fun onDestroy() {
        executor.shutdown()
        super.onDestroy()
    }
}
