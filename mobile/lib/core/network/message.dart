import 'dart:convert';
import 'dart:typed_data';

/// HsoMessage handles binary packet serialization and deserialization
/// for the Knight Online (HSO) protocol using big-endian byte order.
///
/// Opcode commands are signed 8-bit integers (`-128..127`).
class HsoMessage {
  /// Opcode identifying the packet type (-128..127).
  final int command;

  final BytesBuilder _builder = BytesBuilder();
  ByteData? _readData;
  int _readPos = 0;

  /// Converts any integer to signed 8-bit representation (-128..127).
  static int toInt8(int value) =>
      (value & 0x80) != 0 ? (value & 0xFF) - 0x100 : (value & 0xFF);

  /// Creates an empty message to serialize outbound data for [command].
  HsoMessage({required int command}) : command = toInt8(command);

  /// Creates a message initialized with incoming [data] for reading.
  HsoMessage.fromBytes(int command, Uint8List data)
      : command = toInt8(command) {
    final copied = Uint8List.fromList(data);
    _readData = ByteData.sublistView(copied);
  }

  // --- Writers ---

  /// Writes a single 8-bit unsigned byte.
  void writeByte(int value) => _builder.addByte(value & 0xFF);

  /// Writes a boolean flag (1 for true, 0 for false).
  void writeBool(bool value) => writeByte(value ? 1 : 0);

  /// Writes a 16-bit signed integer in big-endian byte order.
  void writeShort(int value) {
    final bd = ByteData(2)..setInt16(0, value, Endian.big);
    _builder.add(bd.buffer.asUint8List());
  }

  /// Writes a 32-bit signed integer in big-endian byte order.
  void writeInt(int value) {
    final bd = ByteData(4)..setInt32(0, value, Endian.big);
    _builder.add(bd.buffer.asUint8List());
  }

  /// Writes a 64-bit signed integer in big-endian byte order.
  void writeLong(int value) {
    final bd = ByteData(8)..setInt64(0, value, Endian.big);
    _builder.add(bd.buffer.asUint8List());
  }

  /// Writes a UTF-8 string prefixed by a 2-byte big-endian length.
  void writeUtf(String text) {
    final bytes = utf8.encode(text);
    writeShort(bytes.length);
    if (bytes.isNotEmpty) {
      _builder.add(bytes);
    }
  }

  /// Appends raw byte slice directly to the payload.
  void writeBytes(List<int> bytes) {
    if (bytes.isNotEmpty) {
      _builder.add(bytes);
    }
  }

  // --- Readers ---

  /// Reads a signed 8-bit integer (-128 to 127).
  int readByte() {
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final val = _readData!.getInt8(_readPos);
    _readPos += 1;
    return val;
  }

  /// Reads an unsigned 8-bit integer (0 to 255).
  int readUnsignedByte() {
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final val = _readData!.getUint8(_readPos);
    _readPos += 1;
    return val;
  }

  /// Reads a boolean value (true if non-zero, false if 0).
  bool readBool() => readByte() != 0;

  /// Reads a 16-bit signed integer in big-endian byte order.
  int readShort() {
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final val = _readData!.getInt16(_readPos, Endian.big);
    _readPos += 2;
    return val;
  }

  /// Reads a 16-bit unsigned integer (0 to 65535) in big-endian byte order.
  int readUnsignedShort() {
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final val = _readData!.getUint16(_readPos, Endian.big);
    _readPos += 2;
    return val;
  }

  /// Reads a 32-bit signed integer in big-endian byte order.
  int readInt() {
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final val = _readData!.getInt32(_readPos, Endian.big);
    _readPos += 4;
    return val;
  }

  /// Reads a 64-bit signed integer in big-endian byte order.
  int readLong() {
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final val = _readData!.getInt64(_readPos, Endian.big);
    _readPos += 8;
    return val;
  }

  /// Reads a UTF-8 string prefixed by a 2-byte big-endian unsigned length.
  String readUtf() {
    final len = readUnsignedShort();
    if (len == 0) return '';
    final bytes = _readData!.buffer.asUint8List(
      _readData!.offsetInBytes + _readPos,
      len,
    );
    _readPos += len;
    return utf8.decode(bytes, allowMalformed: true);
  }

  /// Reads exactly [length] bytes from the payload buffer.
  Uint8List readBytes(int length) {
    if (length <= 0) return Uint8List(0);
    if (_readData == null) {
      throw StateError('Cannot read from an uninitialized payload buffer');
    }
    final bytes = _readData!.buffer.asUint8List(
      _readData!.offsetInBytes + _readPos,
      length,
    );
    _readPos += length;
    return Uint8List.fromList(bytes);
  }

  // --- Buffer Utilities ---

  /// Returns the complete binary payload as a [Uint8List].
  Uint8List toBytes() {
    if (_builder.isNotEmpty) {
      return _builder.toBytes();
    }
    if (_readData != null) {
      return _readData!.buffer.asUint8List(
        _readData!.offsetInBytes,
        _readData!.lengthInBytes,
      );
    }
    return Uint8List(0);
  }

  /// Number of unread bytes remaining in the read buffer.
  int get unreadBytes =>
      _readData != null ? _readData!.lengthInBytes - _readPos : 0;

  /// True if there are remaining unread bytes in the buffer.
  bool get hasRemaining => unreadBytes > 0;

  /// Total length of the message payload in bytes.
  int get length =>
      _readData != null ? _readData!.lengthInBytes : _builder.length;
}
