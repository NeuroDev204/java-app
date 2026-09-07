import Flutter
import UIKit
import AVFoundation
#if canImport(gamebridge)
import gamebridge
#elseif canImport(Gamebridge)
import Gamebridge
#elseif canImport(KnightBridge)
import KnightBridge
#endif

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  private var gameChannel: FlutterMethodChannel?
  // LoadJAR / GetFrameBuffer / Stop block; never run them on the platform thread.
  private let bridgeQueue = DispatchQueue(label: "com.example.knight_online.game_bridge")
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
  // gomobile names the bound type <Package><GoType> — Bridge in package gamebridge.
  private let bridge = GamebridgeBridge()
#endif

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    // Keepalive audioplayers needs a session that stays active in background.
    try? AVAudioSession.sharedInstance().setCategory(.playback, options: [.mixWithOthers])
    try? AVAudioSession.sharedInstance().setActive(true)
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)

    try? AVAudioSession.sharedInstance().setCategory(.playback, options: [.mixWithOthers])
    try? AVAudioSession.sharedInstance().setActive(true)

    let channel = FlutterMethodChannel(
      name: "com.example.knight_online/game_bridge",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    gameChannel = channel
    channel.setMethodCallHandler { [weak self] call, result in
      self?.handleGameBridgeCall(call, result: result)
    }
  }

  private func handleGameBridgeCall(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "loadJAR":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let bytes = (call.arguments as? FlutterStandardTypedData)?.data ?? Data()
      let native = bridge
      bridgeQueue.async {
        let error = native.loadJAR(bytes)
        DispatchQueue.main.async { result(error) }
      }
#else
      result("native bridge not linked")
#endif
    case "getFrameBuffer":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async {
        let buf = native.getFrameBufferIfChanged()
        DispatchQueue.main.async { result(buf) }
      }
#else
      result(nil)
#endif
    case "sendTouchDown":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let args = call.arguments as? [String: Any]
      let x = intArg(args, "x")
      let y = intArg(args, "y")
      let native = bridge
      bridgeQueue.async { native.sendTouchDown(x, y: y) }
#endif
      result(nil)
    case "sendTouchUp":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let args = call.arguments as? [String: Any]
      let x = intArg(args, "x")
      let y = intArg(args, "y")
      let native = bridge
      bridgeQueue.async { native.sendTouchUp(x, y: y) }
#endif
      result(nil)
    case "sendTouchDrag":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let args = call.arguments as? [String: Any]
      let x = intArg(args, "x")
      let y = intArg(args, "y")
      let native = bridge
      bridgeQueue.async { native.sendTouchDrag(x, y: y) }
#endif
      result(nil)
    case "setShouldPaint":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let paint = (call.arguments as? Bool)
        ?? (call.arguments as? NSNumber)?.boolValue
        ?? true
      let native = bridge
      bridgeQueue.async { native.setShouldPaint(paint) }
#endif
      result(nil)
    case "startUpgrade":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let config = call.arguments as? String ?? ""
      let native = bridge
      bridgeQueue.async {
        let error = native.startUpgrade(config)
        DispatchQueue.main.async { result(error) }
      }
#else
      result("native bridge not linked")
#endif
    case "stopUpgrade":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async { native.stopUpgrade() }
#endif
      result(nil)
    case "getUpgradeStatus":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async {
        let status = native.getUpgradeStatus()
        DispatchQueue.main.async { result(status) }
      }
#else
      result(#"{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}"#)
#endif
    case "stop":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async {
        native.stop()
        DispatchQueue.main.async { result(nil) }
      }
#else
      result(nil)
#endif
    case "isStarted":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async {
        let started = native.isStarted()
        DispatchQueue.main.async { result(started) }
      }
#else
      result(false)
#endif
    case "getTextInputInfo":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async {
        let info = native.getTextInputInfoJSON()
        DispatchQueue.main.async { result(info) }
      }
#else
      result(#"{"active":false}"#)
#endif
    case "submitTextInput":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let text = call.arguments as? String ?? ""
      let native = bridge
      bridgeQueue.async { native.submitTextInput(text) }
#endif
      result(nil)
    case "cancelTextInput":
#if canImport(gamebridge) || canImport(Gamebridge) || canImport(KnightBridge)
      let native = bridge
      bridgeQueue.async { native.cancelTextInput() }
#endif
      result(nil)
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  private func intArg(_ args: [String: Any]?, _ key: String) -> Int {
    if let n = args?[key] as? NSNumber {
      return n.intValue
    }
    return args?[key] as? Int ?? 0
  }
}
