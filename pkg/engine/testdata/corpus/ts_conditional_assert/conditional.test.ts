it('conditional assertion', () => {
  const r = parse('x');
  if (r) {
    expect(r.kind).toBe('x');
  }
});
