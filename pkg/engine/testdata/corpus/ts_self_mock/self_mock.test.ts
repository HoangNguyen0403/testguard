it('asserts the mock it just configured', () => {
  const repo = { find: jest.fn().mockReturnValue(42) };
  expect(repo.find()).toBe(42);
});
