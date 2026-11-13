import 'package:dartz/dartz.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

void main() {
  test('repository maps model to entity', () async {
    when(() => dataSource.fetch()).thenAnswer((_) async => Right(fakeUserModel));
    final result = await UserRepository(dataSource).fetch();
    expect(result, Right(fakeUser));
  });
}
