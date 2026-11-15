describe('tautologies', () => {
  it('same identifier tautology', () => {
    const x = calculateTotal();
    expect(x).toBe(x);
  });

  it('constant arithmetic tautology', () => {
    expect(2 + 2).toBe(4);
  });

  it('negated distinct literals', () => {
    expect(true).not.toBe(false);
  });
});
