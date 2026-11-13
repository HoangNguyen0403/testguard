describe('checkout suite', () => {
  it('processes payment', () => {
    const res = pay();
    expect(res).toBe(true);
  });

  it('verifies inventory', () => {
    expect(inStock()).toBe(true);
  });

  it('member call named fit is not a focused test', () => {
    const trained = model.fit([1, 2, 3]);
    expect(trained.loss).toBeLessThan(1);
  });
});
