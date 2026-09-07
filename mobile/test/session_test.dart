import 'dart:typed_data';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/network/crypto.dart';
import 'package:knight_online/core/network/message.dart';
import 'package:knight_online/core/network/session.dart';

void main() {
  group('HsoSession Lifecycle & State', () {
    test('initial state is disconnected with null crypto', () {
      final session = HsoSession();
      expect(session.isConnected, isFalse);
      expect(session.crypto, isNull);
      expect(session.accumulatorLength, equals(0));
    });

    test('send throws StateError when socket is not connected', () {
      final session = HsoSession();
      final msg = HsoMessage(command: 1);
      expect(() => session.send(msg), throwsA(isA<StateError>()));
    });

    test('setCrypto and clearCrypto manage cipher state', () {
      final session = HsoSession();
      final key = Uint8List.fromList([1, 2, 3, 4]);
      session.setCrypto(key);
      expect(session.crypto, isNotNull);
      expect(session.crypto!.keyLength, equals(4));

      session.clearCrypto();
      expect(session.crypto, isNull);
    });

    test('disconnect and dispose cleanly teardown resources', () async {
      final session = HsoSession();
      await session.disconnect();
      expect(session.isConnected, isFalse);
      await session.dispose();
    });
  });

  group('HSO Packet Framing & Encoders', () {
    test('isLargePayloadOpcode recognizes large opcodes', () {
      expect(HsoSession.isLargePayloadOpcode(-51), isTrue);
      expect(HsoSession.isLargePayloadOpcode(-52), isTrue);
      expect(HsoSession.isLargePayloadOpcode(-54), isTrue);
      expect(HsoSession.isLargePayloadOpcode(126), isTrue);
      // Byte values (unsigned 205 is -51 signed)
      expect(HsoSession.isLargePayloadOpcode(205), isTrue);

      // Normal opcodes
      expect(HsoSession.isLargePayloadOpcode(0), isFalse);
      expect(HsoSession.isLargePayloadOpcode(1), isFalse);
      expect(HsoSession.isLargePayloadOpcode(10), isFalse);
      expect(HsoSession.isLargePayloadOpcode(-50), isFalse);
    });

    test('encodePacket standard opcode has 3-byte header [cmd, lenHi, lenLo]',
        () {
      final msg = HsoMessage(command: 10);
      msg.writeByte(42);
      msg.writeByte(43);

      final packet = HsoSession.encodePacket(msg);
      // cmd=10, len=2 (0x00, 0x02), payload=[42, 43]
      expect(packet, equals(Uint8List.fromList([10, 0, 2, 42, 43])));
    });

    test('encodePacket large opcode has 5-byte header [cmd, l0, l1, l2, l3]',
        () {
      final msg = HsoMessage(command: -51);
      msg.writeInt(100);

      final packet = HsoSession.encodePacket(msg);
      // cmd=-51 & 0xFF = 205, len=4 (0x00, 0x00, 0x00, 0x04), payload=4 bytes
      expect(packet.length, equals(9));
      expect(packet[0], equals(205));
      expect(packet[1], equals(0));
      expect(packet[2], equals(0));
      expect(packet[3], equals(0));
      expect(packet[4], equals(4));
    });

    test('encodePacket with crypto encrypts stream sequentially', () {
      final key = Uint8List.fromList([0xAA, 0xBB]);
      final crypto = HsoCrypto(key);

      final msg = HsoMessage(command: 1);
      msg.writeByte(0x10);

      // Plain would be [1, 0, 1, 0x10]
      // Encrypted with key [0xAA, 0xBB]:
      // byte 0: 1 ^ 0xAA = 0xAB
      // byte 1: 0 ^ 0xBB = 0xBB
      // byte 2: 1 ^ 0xAA = 0xAB
      // byte 3: 0x10 ^ 0xBB = 0xAB
      final packet = HsoSession.encodePacket(msg, crypto: crypto);
      expect(packet, equals(Uint8List.fromList([0xAB, 0xBB, 0xAB, 0xAB])));
    });
  });

  group('Stream Reassembly & Accumulator', () {
    test('reassembles single packet received in one chunk', () async {
      final session = HsoSession();
      final received = <HsoMessage>[];
      final sub = session.messages.listen(received.add);

      // Packet: cmd=5, len=2, payload=[11, 22]
      session.handleData([5, 0, 2, 11, 22]);

      // Give stream event microtasks a chance to fire
      await Future<void>.delayed(Duration.zero);
      expect(received.length, equals(1));
      expect(received.first.command, equals(5));
      expect(received.first.readByte(), equals(11));
      expect(received.first.readByte(), equals(22));

      await sub.cancel();
      await session.dispose();
    });

    test('reassembles packet split across multiple TCP chunks', () async {
      final session = HsoSession();
      final received = <HsoMessage>[];
      final sub = session.messages.listen(received.add);

      final msg = HsoMessage(command: 12);
      msg.writeUtf('Hello');
      final fullPacket = HsoSession.encodePacket(msg);

      // Split into 3 arbitrary chunks:
      // Chunk 1: partial header (2 bytes)
      session.handleData(fullPacket.sublist(0, 2));
      await Future<void>.delayed(Duration.zero);
      expect(received, isEmpty);
      expect(session.accumulatorLength, equals(2));

      // Chunk 2: rest of header + partial payload (3 bytes)
      session.handleData(fullPacket.sublist(2, 5));
      await Future<void>.delayed(Duration.zero);
      expect(received, isEmpty);
      expect(session.accumulatorLength, equals(5));

      // Chunk 3: remaining payload bytes
      session.handleData(fullPacket.sublist(5));
      await Future<void>.delayed(Duration.zero);
      expect(received.length, equals(1));
      expect(received.first.command, equals(12));
      expect(received.first.readUtf(), equals('Hello'));
      expect(session.accumulatorLength, equals(0));

      await sub.cancel();
      await session.dispose();
    });

    test('reassembles multiple concatenated packets in a single TCP chunk',
        () async {
      final session = HsoSession();
      final received = <HsoMessage>[];
      final sub = session.messages.listen(received.add);

      final msg1 = HsoMessage(command: 1);
      msg1.writeByte(100);
      final msg2 = HsoMessage(command: 2);
      msg2.writeShort(200);
      final msg3 = HsoMessage(command: 3);
      msg3.writeInt(300);

      final builder = BytesBuilder();
      builder.add(HsoSession.encodePacket(msg1));
      builder.add(HsoSession.encodePacket(msg2));
      builder.add(HsoSession.encodePacket(msg3));

      session.handleData(builder.toBytes());
      await Future<void>.delayed(Duration.zero);

      expect(received.length, equals(3));
      expect(received[0].command, equals(1));
      expect(received[0].readByte(), equals(100));
      expect(received[1].command, equals(2));
      expect(received[1].readShort(), equals(200));
      expect(received[2].command, equals(3));
      expect(received[2].readInt(), equals(300));
      expect(session.accumulatorLength, equals(0));

      await sub.cancel();
      await session.dispose();
    });

    test('reassembles large payload opcode packet (-51)', () async {
      final session = HsoSession();
      final received = <HsoMessage>[];
      final sub = session.messages.listen(received.add);

      final msg = HsoMessage(command: -51);
      msg.writeBytes([1, 2, 3, 4, 5, 6]);
      final packet = HsoSession.encodePacket(msg);

      session.handleData(packet);
      await Future<void>.delayed(Duration.zero);

      expect(received.length, equals(1));
      expect(received.first.command, equals(-51));
      expect(received.first.readBytes(6),
          equals(Uint8List.fromList([1, 2, 3, 4, 5, 6])));

      await sub.cancel();
      await session.dispose();
    });

    test('throws StateError when packet payload length exceeds safety threshold',
        () async {
      final session = HsoSession();
      // Opcode 126 is a 5-byte header opcode.
      // Payload size: 10MB + 1 byte = 10485761 = 0x00A00001
      final oversizedHeader = [126, 0x00, 0xA0, 0x00, 0x01];
      expect(
        () => session.handleData(oversizedHeader),
        throwsA(isA<StateError>().having(
          (e) => e.message,
          'message',
          contains('Packet payload exceeded safety threshold'),
        )),
      );
      expect(session.isConnected, isFalse);
      await session.dispose();
    });
  });

  group('HSO Handshake Protocol', () {
    test('performHandshake configures crypto and processes subsequent stream',
        () async {
      final session = HsoSession();
      final received = <HsoMessage>[];
      final sub = session.messages.listen(received.add);

      final handshakeFuture = session.performHandshake();

      final key = Uint8List.fromList([0x12, 0x34, 0x56, 0x78]);
      // Feed 1-byte length + key bytes
      session.handleData([4, 0x12, 0x34, 0x56, 0x78]);

      await handshakeFuture;
      expect(session.crypto, isNotNull);
      expect(session.crypto!.keyLength, equals(4));

      // Now send an encrypted message from the server
      final serverCrypto = HsoCrypto(key);
      final msg = HsoMessage(command: 7);
      msg.writeUtf('Knight');
      final encPacket = HsoSession.encodePacket(msg, crypto: serverCrypto);

      session.handleData(encPacket);
      await Future<void>.delayed(Duration.zero);

      expect(received.length, equals(1));
      expect(received.first.command, equals(7));
      expect(received.first.readUtf(), equals('Knight'));

      await sub.cancel();
      await session.dispose();
    });

    test('handshake and encrypted payload in single chunk are both parsed',
        () async {
      final session = HsoSession();
      final received = <HsoMessage>[];
      final sub = session.messages.listen(received.add);

      final key = Uint8List.fromList([0xDE, 0xAD, 0xBE, 0xEF]);
      final serverCrypto = HsoCrypto(key);

      final msg = HsoMessage(command: 25);
      msg.writeByte(99);
      final encPacket = HsoSession.encodePacket(msg, crypto: serverCrypto);

      // Combined chunk: [keyLen, key..., encPacket...]
      final combined = BytesBuilder();
      combined.addByte(4);
      combined.add(key);
      combined.add(encPacket);

      final handshakeFuture = session.performHandshake();
      session.handleData(combined.toBytes());

      await handshakeFuture;
      await Future<void>.delayed(Duration.zero);

      expect(session.crypto, isNotNull);
      expect(received.length, equals(1));
      expect(received.first.command, equals(25));
      expect(received.first.readByte(), equals(99));

      await sub.cancel();
      await session.dispose();
    });
  });
}
