describe.only('focused checkout suite', () => {
  it('processes payment', () => {
    const res = pay();
    expect(res).toBe(true);
  });
});

it.only('focused test', () => {
  expect(1 + 1).toBe(2);
});
