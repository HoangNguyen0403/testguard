import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

void main() {
  group('when cache is warm', () {
    setUp(() => when(() => repo.cached()).thenReturn(true));
    test('returns items', () {
      expect(Service(repo).items(), hasLength(2));
    });
  });

  group('when cache is cold', () {
    setUp(() => when(() => repo.cached()).thenReturn(false));
    test('returns items', () {
      expect(Service(repo).items(), hasLength(2));
    });
  });
}
