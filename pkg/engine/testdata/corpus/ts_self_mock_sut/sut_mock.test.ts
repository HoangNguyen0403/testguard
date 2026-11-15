it('asserts SUT behavior using configured mock', () => {
  const repo = { find: jest.fn().mockReturnValue(42) };
  const service = new UserService(repo);
  expect(service.getAnswer()).toBe(42);
});
