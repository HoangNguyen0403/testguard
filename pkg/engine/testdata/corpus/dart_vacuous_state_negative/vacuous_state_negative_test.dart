import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

void main() {
  blocTest<UserBloc, UserState>(
    'asserts specific state properties',
    build: () => UserBloc(),
    act: (bloc) => bloc.add(const LoadUser()),
    expect: () => [
      isA<UserState>().having((s) => s.isLoading, 'isLoading', true),
    ],
  );

  blocTest<UserBloc, UserState>(
    'asserts specific subtypes',
    build: () => UserBloc(),
    act: (bloc) => bloc.add(const LoadUser()),
    expect: () => [
      isA<UserLoading>(),
      isA<UserLoaded>(),
    ],
  );

  blocTest<UserBloc, UserState>(
    'has verify argument',
    build: () => UserBloc(),
    act: (bloc) => bloc.add(const LoadUser()),
    expect: () => [
      isA<UserState>(),
    ],
    verify: (bloc) {
      verify(() => repo.fetchUser()).called(1);
    },
  );
}
