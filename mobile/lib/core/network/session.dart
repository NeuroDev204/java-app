import 'dart:async';
import 'dart:io';
import 'dart:typed_data';
import 'crypto.dart';
import 'message.dart';

/// HsoSession manages the TCP socket connection, stream cipher state,
/// and message framing for network communication with the HSO game server.
///
/// Implements length-prefixed framing and stream reassembly matching
/// the TeaMobi JVM client and Go server protocol (`pkg/network/protocol`).
class HsoSession {
  Socket? _socket;
  StreamSubscription<List<int>>? _subscription;
  HsoCrypto? _crypto;
  final StreamController<HsoMessage> _messageController =
      StreamController<HsoMessage>.broadcast();

  List<int> _accumulator = <int>[];
  Completer<void>? _handshakeCompleter;
  bool _handshakePending = false;

  HsoSession({Socket? socket, HsoCrypto? crypto})
      : _socket = socket,
        _crypto = crypto {
    if (_socket != null) {
      _subscription = _socket!.listen(
        _onData,
        onError: _onError,
        onDone: _onDone,
      );
    }
  }

  /// Broadcast stream emitting incoming deserialized [HsoMessage] packets.
  Stream<HsoMessage> get messages => _messageController.stream;

  /// True if the underlying TCP socket is currently connected.
  bool get isConnected => _socket != null;

  /// The active stream cipher, or null if unencrypted.
  HsoCrypto? get crypto => _crypto;

  /// Current unparsed bytes held in the accumulator (for testing / diagnostics).
  int get accumulatorLength => _accumulator.length;

  /// Returns true if the opcode requires a 4-byte big-endian length header
  /// instead of the standard 2-byte header.
  ///
  /// In HSO: -51, -52, -54, 126.
  static bool isLargePayloadOpcode(int cmd) {
    final signedCmd = HsoMessage.toInt8(cmd);
    return signedCmd == -51 ||
        signedCmd == -52 ||
        signedCmd == -54 ||
        signedCmd == 126;
  }

  /// Encodes and frames an [HsoMessage] into raw network bytes.
  /// If [crypto] is provided, headers and payload are encrypted in stream order.
  static Uint8List encodePacket(HsoMessage msg, {HsoCrypto? crypto}) {
    final cmd = msg.command;
    final payload = msg.toBytes();
    final size = payload.length;
    final isLarge = isLargePayloadOpcode(cmd);

    if (crypto != null) {
      final encCmd = crypto.encryptByte(cmd & 0xFF);
      final List<int> header;
      if (isLarge) {
        header = <int>[
          encCmd,
          crypto.encryptByte((size >> 24) & 0xFF),
          crypto.encryptByte((size >> 16) & 0xFF),
          crypto.encryptByte((size >> 8) & 0xFF),
          crypto.encryptByte(size & 0xFF),
        ];
      } else {
        header = <int>[
          encCmd,
          crypto.encryptByte((size >> 8) & 0xFF),
          crypto.encryptByte(size & 0xFF),
        ];
      }
      final encPayload = crypto.encrypt(payload);
      final packet = Uint8List(header.length + encPayload.length);
      packet.setRange(0, header.length, header);
      packet.setRange(header.length, packet.length, encPayload);
      return packet;
    } else {
      final List<int> header;
      if (isLarge) {
        header = <int>[
          cmd & 0xFF,
          (size >> 24) & 0xFF,
          (size >> 16) & 0xFF,
          (size >> 8) & 0xFF,
          size & 0xFF,
        ];
      } else {
        header = <int>[
          cmd & 0xFF,
          (size >> 8) & 0xFF,
          size & 0xFF,
        ];
      }
      final packet = Uint8List(header.length + payload.length);
      packet.setRange(0, header.length, header);
      packet.setRange(header.length, packet.length, payload);
      return packet;
    }
  }

  /// Connects to the specified [host] and [port] via TCP.
  Future<void> connect(
    String host,
    int port, {
    Duration timeout = const Duration(seconds: 10),
  }) async {
    await disconnect();
    final socket = await Socket.connect(host, port, timeout: timeout);
    _socket = socket;
    _subscription = socket.listen(
      _onData,
      onError: _onError,
      onDone: _onDone,
      cancelOnError: true,
    );
  }

  /// Performs the initial HSO protocol handshake.
  /// Reads 1-byte key length followed by raw key bytes, and initializes [HsoCrypto].
  Future<void> performHandshake({
    Duration timeout = const Duration(seconds: 10),
  }) async {
    if (_crypto != null) return;
    if (_handshakeCompleter != null) {
      return _handshakeCompleter!.future;
    }

    _handshakePending = true;
    final completer = Completer<void>();
    _handshakeCompleter = completer;

    // Check if handshake data has already accumulated
    _checkHandshake();

    return completer.future.timeout(
      timeout,
      onTimeout: () {
        _handshakePending = false;
        _handshakeCompleter = null;
        throw TimeoutException('Handshake timed out after $timeout');
      },
    );
  }

  /// Configures or updates the stream XOR cipher with the given [key].
  void setCrypto(Uint8List key) {
    _crypto = HsoCrypto(key);
  }

  /// Clears the stream cipher, reverting to plain/unencrypted traffic.
  void clearCrypto() {
    _crypto = null;
  }

  /// Serializes, frames, and sends an [HsoMessage] over the network stream.
  /// Throws [StateError] if the socket is not connected.
  void send(HsoMessage msg) {
    if (_socket == null) {
      throw StateError('Cannot send message: session is not connected');
    }
    final packet = encodePacket(msg, crypto: _crypto);
    _socket!.add(packet);
  }

  /// Ingests incoming data chunks from the network or test harness.
  void handleData(List<int> data) => _onData(data);

  void _onData(List<int> data) {
    if (data.isEmpty || _messageController.isClosed) return;

    if (_handshakePending) {
      _accumulator.addAll(data);
      _checkHandshake();
      return;
    }

    if (_crypto != null) {
      final dec = Uint8List(data.length);
      for (int i = 0; i < data.length; i++) {
        dec[i] = _crypto!.decryptByte(data[i]);
      }
      _accumulator.addAll(dec);
    } else {
      _accumulator.addAll(data.map((b) => b & 0xFF));
    }

    _processAccumulator();
  }

  void _checkHandshake() {
    if (_accumulator.isEmpty) return;
    final keyLen = _accumulator[0];
    if (keyLen <= 0) {
      final error = StateError('Invalid handshake key length: $keyLen');
      _handshakePending = false;
      final completer = _handshakeCompleter;
      _handshakeCompleter = null;
      completer?.completeError(error);
      return;
    }

    if (_accumulator.length < 1 + keyLen) {
      // Need more key bytes
      return;
    }

    final key = Uint8List.fromList(_accumulator.sublist(1, 1 + keyLen));
    final remainingRaw = _accumulator.sublist(1 + keyLen);
    _accumulator = <int>[];
    setCrypto(key);
    _handshakePending = false;
    final completer = _handshakeCompleter;
    _handshakeCompleter = null;
    completer?.complete();

    // If more bytes arrived in the same chunk after handshake, process them
    if (remainingRaw.isNotEmpty) {
      _onData(remainingRaw);
    }
  }

  void _processAccumulator() {
    if (_messageController.isClosed) return;

    while (true) {
      if (_accumulator.isEmpty) break;

      final cmdRaw = _accumulator[0];
      final cmd = HsoMessage.toInt8(cmdRaw);
      final isLarge = isLargePayloadOpcode(cmd);
      final headerSize = isLarge ? 5 : 3;

      if (_accumulator.length < headerSize) {
        // Need more bytes for header
        break;
      }

      final int payloadLen;
      if (isLarge) {
        payloadLen = (_accumulator[1] << 24) |
            (_accumulator[2] << 16) |
            (_accumulator[3] << 8) |
            _accumulator[4];
      } else {
        payloadLen = (_accumulator[1] << 8) | _accumulator[2];
      }

      if (payloadLen < 0) {
        disconnect();
        throw StateError(
          'Invalid negative payload length ($payloadLen) for opcode $cmd',
        );
      }

      if (payloadLen > 10 * 1024 * 1024) {
        disconnect();
        throw StateError(
          'Packet payload exceeded safety threshold: $payloadLen bytes',
        );
      }

      final totalSize = headerSize + payloadLen;
      if (_accumulator.length < totalSize) {
        // Need more bytes for complete payload
        break;
      }

      final payload = Uint8List.fromList(
        _accumulator.sublist(headerSize, totalSize),
      );
      _accumulator = _accumulator.sublist(totalSize);

      _messageController.add(HsoMessage.fromBytes(cmd, payload));
    }
  }

  void _onError(Object error) {
    disconnect();
  }

  void _onDone() {
    disconnect();
  }

  /// Closes the TCP connection and resets the cipher state and accumulator.
  Future<void> disconnect() async {
    try {
      await _subscription?.cancel();
    } catch (_) {}
    _subscription = null;

    try {
      await _socket?.close();
    } catch (_) {}
    _socket = null;
    _crypto?.reset();
    _accumulator.clear();
    _handshakePending = false;
    if (_handshakeCompleter != null && !_handshakeCompleter!.isCompleted) {
      _handshakeCompleter!.completeError(
        StateError('Session disconnected during handshake'),
      );
      _handshakeCompleter = null;
    }
  }

  /// Disconnects and terminates the internal message stream controller.
  Future<void> dispose() async {
    await disconnect();
    await _messageController.close();
  }
}
