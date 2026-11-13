import 'package:dartz/dartz.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

void main() {
  test('pass-through use case', () async {
    when(() => repo.fetch()).thenAnswer((_) async => Right(fakeUser));
    final result = await GetUser(repo)();
    expect(result, Right(fakeUser));
  });
}
