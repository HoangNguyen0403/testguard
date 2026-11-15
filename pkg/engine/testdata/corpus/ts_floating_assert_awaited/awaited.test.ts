it('awaited async assertion', async () => {
  await expect(fetchUser('a')).resolves.toEqual({ id: 'a' });
});
