import 'package:test/test.dart';

void main() {
  test('calculates net price with non-zero discount', () {
    final discount = 10.0;
    final total = calculateNetPrice(price: 100.0, discount: discount);
    expect(total, equals(90.0));
  });

  test('boundary: calculates base price when zero discount applied', () {
    final discount = 0.0;
    final total = calculateNetPrice(price: 100.0, discount: discount);
    expect(total, equals(100.0));
  });
}
