import 'package:flutter_test/flutter_test.dart';

void main() {
  test('A4 real test', () {
    expect(add(1, 2), 3);
  });

  // test('A8 line commented', () {});
  /* test('A9 block commented', () {}, skip: true); */
  /// test('A10 doc commented', () {});
  // blocTest<Cubit, State>('A11 commented bloc', build: () => Cubit(), expect: () => []);
  // test('A12 commented unbalanced', () {
  //   expect(1, 1);
  // });

  test('L17 json fixture', () {
    const json = '''{"name": "a", "tags": ["x"]}''';
    expect(decode(json).name, 'a');
  });

  test('L22 after', () {
    expect(add(2, 2), 4);
  });
}
