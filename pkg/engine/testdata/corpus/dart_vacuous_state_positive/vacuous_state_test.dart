import 'package:bloc_test/bloc_test.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  blocTest<UserBloc, UserState>(
    'vacuous state passes for any emitted state',
    build: () => UserBloc(),
    act: (bloc) => bloc.add(const LoadUser()),
    expect: () => [
      isA<UserState>(),
    ],
  );
}
