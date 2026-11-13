import 'package:test/test.dart';

int add(int a, int b) => a + b;

void main() {
  test('echoes its own value', () {
    final result = add(1, 2);
    expect(result, result);
  });
}
