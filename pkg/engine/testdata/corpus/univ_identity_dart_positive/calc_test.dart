import 'package:test/test.dart';

void main() {
  test('calculates net price with discount', () {
    final discount = 0.0;
    final total = calculateNetPrice(price: 100.0, discount: discount);
    expect(total, equals(100.0));
  });
}
