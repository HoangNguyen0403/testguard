const add = (a: number, b: number): number => a + b;

describe('add', () => {
  it('adds one and one', () => {
    expect(add(1, 1)).toBe(2);
  });

  it('adds two and two', () => {
    expect(add(2, 2)).toBe(4);
  });

  it('adds three and three', () => {
    expect(add(3, 3)).toBe(6);
  });

  it('adds four and four', () => {
    expect(add(4, 4)).toBe(8);
  });
});
