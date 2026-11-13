void main() {
  blocTest<SearchBloc, SearchState>(
    'search emits loading',
    build: () => SearchBloc(),
    act: (bloc) => bloc.add(const SearchEvent('key')),
    expect: () => [SearchState.loading()],
  );

  blocTest<SearchBloc, SearchState>(
    'search emits loading again',
    build: () => SearchBloc(),
    act: (bloc) => bloc.add(const SearchEvent('key')),
    expect: () => [SearchState.loading()],
  );
}
