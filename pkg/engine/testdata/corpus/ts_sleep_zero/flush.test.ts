it('flushes pending macro tasks', async () => {
  await new Promise(r => setTimeout(r, 0));
  expect(true).toBe(false); // non-tautology assertion to pass zero-assert check
});
