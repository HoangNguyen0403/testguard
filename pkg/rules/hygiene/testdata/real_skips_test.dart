import 'package:flutter_test/flutter_test.dart';

// Regression fixture for GUARD-UNIV-SKIP: a real skip directive must still be
// reported at its own line.
void main() {
  test('flaky', () {}, skip: true);

  test('real', () {
    expect(add(1, 2), 3);
  });
}
