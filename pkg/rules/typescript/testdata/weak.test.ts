import { add, fetchUser, parse } from './sut';

describe('weak oracles an agent commonly emits', () => {
  it('bare expect without matcher', () => {
    const r = add(1, 2);
    expect(r);
  });

  it('existence only', () => {
    const r = add(1, 2);
    expect(r).toBeDefined();
  });

  it('swallowed error path', async () => {
    try {
      await fetchUser('missing');
    } catch (e) {
      expect(e).toBeInstanceOf(Error);
    }
  });

  it('floating async assertion', () => {
    expect(fetchUser('a')).resolves.toEqual({ id: 'a' });
  });

  it('conditional assertion', () => {
    const r = parse('x');
    if (r) {
      expect(r.kind).toBe('x');
    }
  });

  it('asserts the mock it just configured', () => {
    const repo = { find: jest.fn().mockReturnValue(42) };
    expect(repo.find()).toBe(42);
  });

  it.only('focused test left behind', () => {
    expect(add(1, 2)).toBe(3);
  });

  // testguard:ignore ALL
  it('suppressed empty test', () => {
    add(1, 2);
  });

  it.each([[1, 2, 3]])('parameterised %i', (a, b, c) => {
    add(a, b);
  });

  it(`template literal name`, () => {
    add(1, 2);
  });
});
