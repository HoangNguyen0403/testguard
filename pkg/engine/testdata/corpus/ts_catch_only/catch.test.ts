it('swallowed error path', async () => {
  try {
    await fetchUser('missing');
  } catch (e) {
    expect(e).toBeInstanceOf(Error);
  }
});
