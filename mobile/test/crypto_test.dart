import 'dart:typed_data';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/network/crypto.dart';

void main() {
  test('HsoCrypto encrypt and decrypt roundtrip matches HSO algorithm', () {
    final key = Uint8List.fromList([0x12, 0x34, 0x56, 0x78]);
    final encCrypto = HsoCrypto(key);
    final decCrypto = HsoCrypto(key);

    final original = Uint8List.fromList([10, 20, 30, 40, 50, 60]);
    final encrypted = Uint8List(original.length);
    for (int i = 0; i < original.length; i++) {
      encrypted[i] = encCrypto.encryptByte(original[i]);
    }

    final decrypted = Uint8List(original.length);
    for (int i = 0; i < original.length; i++) {
      decrypted[i] = decCrypto.decryptByte(encrypted[i]);
    }

    expect(decrypted, equals(original));
  });

  test('HsoCrypto pointer wraps around key length', () {
    final key = Uint8List.fromList([0xAA, 0xBB]);
    final crypto = HsoCrypto(key);

    expect(crypto.encryptByte(0x00), equals(0xAA));
    expect(crypto.encryptByte(0x00), equals(0xBB));
    expect(crypto.encryptByte(0x00), equals(0xAA)); // Wrapped
  });

  test('HsoCrypto bulk encrypt and decrypt methods', () {
    final key = Uint8List.fromList([0x01, 0x02, 0x03]);
    final encCrypto = HsoCrypto(key);
    final decCrypto = HsoCrypto(key);

    final original = Uint8List.fromList([100, 101, 102, 103, 104]);
    final encrypted = encCrypto.encrypt(original);
    final decrypted = decCrypto.decrypt(encrypted);

    expect(decrypted, equals(original));
  });

  test('HsoCrypto reset restores read and send stream pointers', () {
    final key = Uint8List.fromList([0x55, 0xAA]);
    final crypto = HsoCrypto(key);

    expect(crypto.encryptByte(0x00), equals(0x55));
    expect(crypto.decryptByte(0x00), equals(0x55));
    expect(crypto.sendPointer, equals(1));
    expect(crypto.readPointer, equals(1));

    crypto.reset();
    expect(crypto.sendPointer, equals(0));
    expect(crypto.readPointer, equals(0));
    expect(crypto.encryptByte(0x00), equals(0x55));
    expect(crypto.decryptByte(0x00), equals(0x55));
  });

  test('HsoCrypto handles empty key without modification', () {
    final crypto = HsoCrypto(Uint8List(0));
    expect(crypto.encryptByte(42), equals(42));
    expect(crypto.decryptByte(42), equals(42));
    final data = Uint8List.fromList([1, 2, 3]);
    expect(crypto.encrypt(data), equals(data));
    expect(crypto.decrypt(data), equals(data));
  });
}
