import { sum } from './sut';

describe('nested template with literal brace', () => {
  it('L4 first', () => {
    const s = `a ${[1].map((n) => `${n}}`).join('')} b`;
    expect(s).toContain('1}');
  });

  it('L9 second', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it('L13 third', () => {
    expect(sum(2, 2)).toBe(4);
  });
});
