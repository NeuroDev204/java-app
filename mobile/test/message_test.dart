import 'dart:typed_data';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/network/message.dart';

void main() {
  test('HsoMessage read/write primitives correctly serialize', () {
    final msg = HsoMessage(command: 10);
    msg.writeByte(42);
    msg.writeShort(1234);
    msg.writeInt(987654);
    msg.writeUtf('KnightOnline');

    final reader = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader.command, equals(10));
    expect(reader.readByte(), equals(42));
    expect(reader.readShort(), equals(1234));
    expect(reader.readInt(), equals(987654));
    expect(reader.readUtf(), equals('KnightOnline'));
  });

  test('HsoMessage handles boolean, long, and byte arrays', () {
    final msg = HsoMessage(command: 25);
    msg.writeBool(true);
    msg.writeBool(false);
    msg.writeLong(1234567890123456);
    msg.writeBytes([1, 2, 3, 4, 5]);

    final reader = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader.readBool(), isTrue);
    expect(reader.readBool(), isFalse);
    expect(reader.readLong(), equals(1234567890123456));
    final bytes = reader.readBytes(5);
    expect(bytes, equals(Uint8List.fromList([1, 2, 3, 4, 5])));
  });

  test('HsoMessage reads unsigned bytes and shorts correctly', () {
    final msg = HsoMessage(command: 5);
    msg.writeByte(200); // 0xC8 -> -56 signed, 200 unsigned
    msg.writeShort(50000); // 0xC350 -> -15536 signed, 50000 unsigned

    final reader = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader.readByte(), equals(-56));
    expect(reader.readShort(), equals(-15536));

    final reader2 = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader2.readUnsignedByte(), equals(200));
    expect(reader2.readUnsignedShort(), equals(50000));
  });

  test('HsoMessage normalizes opcodes to signed int8 (-128..127)', () {
    expect(HsoMessage(command: 10).command, equals(10));
    expect(HsoMessage(command: -51).command, equals(-51));
    // 205 unsigned byte is -51 signed
    expect(HsoMessage(command: 205).command, equals(-51));
    expect(HsoMessage(command: 255).command, equals(-1));
    expect(HsoMessage(command: 128).command, equals(-128));

    final reader = HsoMessage.fromBytes(205, Uint8List(0));
    expect(reader.command, equals(-51));
  });

  test('HsoMessage readUtf handles empty strings and malformed UTF-8 gracefully', () {
    final msg = HsoMessage(command: 1);
    msg.writeUtf('');
    final reader = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader.readUtf(), equals(''));

    // Malformed UTF-8 payload prefixed with 2-byte length
    final malformedPayload = Uint8List.fromList([0, 2, 0xFF, 0xFE]);
    final malformedReader = HsoMessage.fromBytes(1, malformedPayload);
    // allowMalformed: true replaces invalid sequences without throwing
    expect(malformedReader.readUtf(), contains('\uFFFD'));
  });

  test('HsoMessage tracks remaining bytes and length accurately', () {
    final msg = HsoMessage(command: 1);
    msg.writeInt(100);
    msg.writeInt(200);

    final reader = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader.length, equals(8));
    expect(reader.unreadBytes, equals(8));
    expect(reader.hasRemaining, isTrue);

    reader.readInt();
    expect(reader.unreadBytes, equals(4));
    expect(reader.hasRemaining, isTrue);

    reader.readInt();
    expect(reader.unreadBytes, equals(0));
    expect(reader.hasRemaining, isFalse);
  });
}
