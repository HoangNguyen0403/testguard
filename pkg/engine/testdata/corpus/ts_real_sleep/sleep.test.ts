it('sleeps arbitrarily in unit test', async () => {
  await new Promise(r => setTimeout(r, 1000));
  expect(data.isReady).toBe(true);
});
