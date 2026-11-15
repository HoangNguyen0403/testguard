import 'package:test/test.dart';

int add(int a, int b) => a + b;

void main() {
  test('foo', () {
    expect(add(1, 2), 3);
  });
}
