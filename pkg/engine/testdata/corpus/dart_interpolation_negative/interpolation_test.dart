import 'package:flutter_test/flutter_test.dart';

void main() {
  test('B4 raw string with fake test', () {
    final raw = r'test("B5 fake", () {});';
    expect(raw.length, 22);
  });

  test('B9 triple quoted with fake test and skip', () {
    const source = '''
      test('B11 fake', () {});
      skip: true
    ''';
    expect(source.length, 50);
  });

  test('B17 interpolation with brace key', () {
    final m = {'}': 1};
    final s = '${m['}']} end';
    expect(s, '1 end');
  });

  test('B23 after interpolation', () {
    expect(add(1, 2), 3);
  });

  test('B27 plain string with brace', () {
    final s = 'closing } brace';
    expect(s.length, 15);
  });

  test('B32 escaped dollar', () {
    final s = '\${not} interpolated';
    expect(s.length, 19);
  });

  test('B37 nested interpolation string', () {
    final s = '${'inner ${x}'} end';
    expect(s, 'inner 1 end');
  });

  test('B42 interpolation with closure braces', () {
    final s = '${(() { return 1; })()}';
    expect(s, '1');
  });
}
