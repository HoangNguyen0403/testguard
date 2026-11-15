it('unconditional assertion with branch', () => {
  const r = parse('x');
  expect(r).toBeDefined();
  if (r) {
    expect(r.kind).toBe('x');
  }
});
