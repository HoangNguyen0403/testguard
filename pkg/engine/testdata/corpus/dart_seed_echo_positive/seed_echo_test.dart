import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  blocTest<UserBloc, UserState>(
    'seed echo positive',
    build: () => UserBloc(),
    seed: () => UserState(name: 'bob', age: 3),
    act: (bloc) => bloc.add(const RefreshEvent()),
    expect: () => [
      isA<UserState>().having((s) => s.name, 'name', 'bob'),
    ],
  );
}
