import 'package:test/test.dart';

int add(int a, int b) => a + b;

void main() {
  test('adds one and one', () {
    expect(add(1, 1), 2);
  });

  test('adds two and two', () {
    expect(add(2, 2), 4);
  });

  test('adds three and three', () {
    expect(add(3, 3), 6);
  });

  test('adds four and four', () {
    expect(add(4, 4), 8);
  });
}
