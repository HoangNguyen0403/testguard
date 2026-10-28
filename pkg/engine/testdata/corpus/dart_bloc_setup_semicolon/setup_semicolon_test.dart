void main() {
  blocTest<SearchBloc, SearchState>(
    'search with failure A',
    setUp: () {
      when(() => repo.get()).thenAnswer((_) async => const Left(ApiFailure.other('e')));
      when(() => repo.fallbackA()).thenReturn(1);
    },
    build: () => SearchBloc(repo),
    act: (bloc) => bloc.add(const SearchEvent('a')),
    expect: () => [SearchState.failureA()],
  );

  blocTest<SearchBloc, SearchState>(
    'search with failure B',
    setUp: () {
      when(() => repo.get()).thenAnswer((_) async => const Left(ApiFailure.other('e')));
      when(() => repo.fallbackB()).thenReturn(2);
    },
    build: () => SearchBloc(repo),
    act: (bloc) => bloc.add(const SearchEvent('b')),
    expect: () => [SearchState.failureB()],
  );
}
