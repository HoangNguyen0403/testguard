describe('math operations', () => {
  it('adds small numbers', () => {
    expect(add(1, 2)).toBe(3);
  });

  it('adds zero', () => {
    expect(add(0, 0)).toBe(0);
  });
});
