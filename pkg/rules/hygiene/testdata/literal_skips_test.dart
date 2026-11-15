import 'package:flutter_test/flutter_test.dart';

// Regression fixture for GUARD-UNIV-SKIP: skip text that appears only inside a
// raw, triple-quoted, or interpolated string is data, not a skipped test, so
// this file must produce no findings.
void main() {
  test('raw string with fake test', () {
    const raw = r'test("fake", () {}, skip: true);';
    expect(raw.length, 32);
  });

  test('triple quoted with fake skip', () {
    const source = '''
      test('fake', () {});
      skip: true
    ''';
    expect(source.length, 48);
  });

  test('interpolation with brace key', () {
    final m = {'}': 1};
    final s = '${m['}']} @Skip("fake")';
    expect(s, '1 @Skip("fake")');
  });
}
