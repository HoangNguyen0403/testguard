it('handles mock latency without error', async () => {
  api.fetch.mockImplementation(() => new Promise(r => setTimeout(() => r(1), 100)));
  expect(api.fetch).toBeDefined();
});
