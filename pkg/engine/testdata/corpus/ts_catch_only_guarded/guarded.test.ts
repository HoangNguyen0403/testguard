it('guarded error path', async () => {
  try {
    await fetchUser('missing');
    fail('expected function to throw');
  } catch (e) {
    expect(e).toBeInstanceOf(Error);
  }
});
