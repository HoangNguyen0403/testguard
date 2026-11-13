import 'package:test/test.dart';

int add(int a, int b) => a + b;

void main() {
  test('checks addition', () {
    expect(add(1, 2), 3);
  }, skip: true);
}
