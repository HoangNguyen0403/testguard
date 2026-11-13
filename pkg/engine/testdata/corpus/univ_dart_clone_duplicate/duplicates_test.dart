import 'package:test/test.dart';

int add(int a, int b) => a + b;

void main() {
  test('adds two numbers', () {
    expect(add(1, 2), 3);
  });

  test('adds two numbers again', () {
    expect(add(1, 2), 3);
  });
}
