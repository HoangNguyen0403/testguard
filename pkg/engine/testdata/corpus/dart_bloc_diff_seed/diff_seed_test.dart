void main() {
  blocTest<SearchBloc, SearchState>(
    'search with initial state',
    seed: () => SearchState.initial(),
    build: () => SearchBloc(),
    act: (bloc) => bloc.add(const SearchEvent('key')),
    expect: () => [SearchState.loading()],
  );

  blocTest<SearchBloc, SearchState>(
    'search with custom state',
    seed: () => SearchState.ready(),
    build: () => SearchBloc(),
    act: (bloc) => bloc.add(const SearchEvent('key')),
    expect: () => [SearchState.loading()],
  );
}
