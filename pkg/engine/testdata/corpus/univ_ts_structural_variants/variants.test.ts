const add = (a: number, b: number): number => a + b;

describe('add', () => {
  it('adds one and one', () => {
    const result = add(1, 1);
    expect(result).toBe(2);
  });

  it('adds two and two', () => {
    const result = add(2, 2);
    expect(result).toBe(4);
  });

  it('adds three and three', () => {
    const result = add(3, 3);
    expect(result).toBe(6);
  });

  it('adds four and four', () => {
    const result = add(4, 4);
    expect(result).toBe(8);
  });
});
