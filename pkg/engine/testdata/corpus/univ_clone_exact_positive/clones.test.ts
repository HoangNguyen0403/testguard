describe('billing', () => {
  it('charges customer card', () => {
    const res = charge(100);
    expect(res.success).toBe(true);
  });

  it('charges customer card duplicate', () => {
    const res = charge(100);
    expect(res.success).toBe(true);
  });
});
