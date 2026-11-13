describe('valid assertions', () => {
  it('asserts against next invocation', () => {
    expect(generator.next()).toBe(generator.next());
  });

  it('asserts real state against expected', () => {
    const result = calculateTotal();
    expect(result).toBe(42);
  });
});
