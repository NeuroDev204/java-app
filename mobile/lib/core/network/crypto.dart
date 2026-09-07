import 'dart:typed_data';

/// HsoCrypto implements symmetric stream XOR encryption and decryption
/// using the TeaMobi Knight Online (HSO) dynamic key algorithm.
///
/// It maintains independent pointers for reading (decryption, `r`) and
/// sending (encryption, `s`), exactly matching the client and server protocol.
class HsoCrypto {
  final Uint8List _key;
  int _r = 0;
  int _s = 0;

  HsoCrypto(Uint8List key) : _key = Uint8List.fromList(key);

  /// Decrypts a single byte from the stream and advances the read pointer (_r).
  /// Formula: res = (key[r] ^ b) & 0xFF; r = (r + 1) % len(key)
  int decryptByte(int b) {
    if (_key.isEmpty) return b & 0xFF;
    final res = _key[_r] ^ b;
    _r = (_r + 1) % _key.length;
    return res & 0xFF;
  }

  /// Encrypts a single byte in the stream and advances the send pointer (_s).
  /// Formula: res = (key[s] ^ b) & 0xFF; s = (s + 1) % len(key)
  int encryptByte(int b) {
    if (_key.isEmpty) return b & 0xFF;
    final res = _key[_s] ^ b;
    _s = (_s + 1) % _key.length;
    return res & 0xFF;
  }

  /// Encrypts an entire byte buffer in stream order and advances the send pointer.
  Uint8List encrypt(Uint8List src) {
    if (_key.isEmpty || src.isEmpty) return Uint8List.fromList(src);
    final dst = Uint8List(src.length);
    for (int i = 0; i < src.length; i++) {
      dst[i] = encryptByte(src[i]);
    }
    return dst;
  }

  /// Decrypts an entire byte buffer in stream order and advances the read pointer.
  Uint8List decrypt(Uint8List src) {
    if (_key.isEmpty || src.isEmpty) return Uint8List.fromList(src);
    final dst = Uint8List(src.length);
    for (int i = 0; i < src.length; i++) {
      dst[i] = decryptByte(src[i]);
    }
    return dst;
  }

  /// Resets both the decryption (_r) and encryption (_s) stream pointers to 0.
  void reset() {
    _r = 0;
    _s = 0;
  }

  /// Current read (decryption) pointer position.
  int get readPointer => _r;

  /// Current send (encryption) pointer position.
  int get sendPointer => _s;

  /// Length of the dynamic cipher key.
  int get keyLength => _key.length;
}
