import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  blocTest<UserBloc, UserState>(
    'keeps seeded user when refresh fails',
    build: () => UserBloc(),
    seed: () => UserState(user: User(name: 'alice'), status: Status.idle),
    act: (bloc) => bloc.add(const RefreshFailed()),
    expect: () => [
      isA<UserState>()
          .having((s) => s.status, 'status', Status.error)
          .having((s) => s.user, 'user', User(name: 'alice')),
    ],
  );

  blocTest<UserBloc, UserState>(
    'nested seed field is not a state field',
    build: () => UserBloc(),
    seed: () => UserState(user: User(name: 'alice'), name: 'bob'),
    act: (bloc) => bloc.add(const Renamed('alice')),
    expect: () => [
      isA<UserState>().having((s) => s.name, 'name', 'alice'),
    ],
  );
}
