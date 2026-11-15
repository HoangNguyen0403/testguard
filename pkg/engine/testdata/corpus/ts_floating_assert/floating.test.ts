it('floating async assertion', () => {
  expect(fetchUser('a')).resolves.toEqual({ id: 'a' });
});
