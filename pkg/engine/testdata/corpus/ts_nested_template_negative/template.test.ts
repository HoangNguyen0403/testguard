import { sum } from './sut';

describe('nested template with literal brace', () => {
  it('nested interpolation keeps first test', () => {
    const s = `a ${[1].map((n) => `${n}}`).join('')} b`;
    expect(s).toContain('1}');
  });

  it('second test still discovered', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it('third test still discovered', () => {
    expect(sum(2, 2)).toBe(4);
  });
});
