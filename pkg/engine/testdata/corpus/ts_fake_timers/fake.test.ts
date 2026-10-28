beforeEach(() => {
  jest.useFakeTimers();
});

it('advances timer with fake timers', () => {
  setTimeout(() => {}, 500);
  expect(true).toBe(false);
});
