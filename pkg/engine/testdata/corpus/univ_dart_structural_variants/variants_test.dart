import 'package:test/test.dart';

int add(int a, int b) => a + b;

void main() {
  test('adds one and one', () {
    final result = add(1, 1);
    expect(result, 2);
  });

  test('adds two and two', () {
    final result = add(2, 2);
    expect(result, 4);
  });

  test('adds three and three', () {
    final result = add(3, 3);
    expect(result, 6);
  });

  test('adds four and four', () {
    final result = add(4, 4);
    expect(result, 8);
  });
}
