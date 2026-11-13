import { svc, load, parse, sum } from './sut';

describe('hollow tests, finding expected on each', () => {
  it('L4 ZERO only logs', () => {
    console.log(sum(1, 2));
  });

  it('L8 ZERO expect shadowed by local stub', () => {
    const expect = (v: unknown) => ({ toBe: (_: unknown) => v });
    expect(sum(1, 2)).toBe(3);
  });

  it('L13 ZERO helper with same name as asserting helper in other suite', () => {
    verify(sum(1, 2));
  });

  it('L17 TAUT literal', () => {
    expect(true).toBe(true);
  });

  it('L21 TAUT same ident', () => {
    const r = sum(1, 2);
    expect(r).toEqual(r);
  });

  it('L26 FLOATING resolves without await in async', async () => {
    expect(load('x')).resolves.toEqual({ id: 1 });
  });

  it('L30 CONDITIONAL braceless if', () => {
    const r = parse('x');
    if (r.ok) expect(r.value).toBe('x');
  });

  it('L35 CONDITIONAL else only logs', () => {
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      console.log('skip');
    }
  });

  it('L44 CATCH-ONLY async', async () => {
    try {
      await load('bad');
    } catch (e) {
      expect(e).toBeInstanceOf(Error);
    }
  });

  it('L52 SELF-MOCK return value echoed', () => {
    const repo = { find: jest.fn().mockReturnValue(42) };
    expect(repo.find()).toBe(42);
  });

  it('L57 EXISTENCE only', () => {
    expect(svc).toBeDefined();
  });

  it('L61 ZERO assertion only in nested never-called function', () => {
    function unused() {
      expect(sum(1, 2)).toBe(3);
    }
    sum(1, 2);
  });

  xit('L68 SKIP xit', () => {
    expect(sum(1, 2)).toBe(3);
  });

  fit('L72 FOCUS fit', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it('L76 NAME test1', () => {
    expect(sum(1, 2)).toBe(3);
  });
});

describe('other suite', () => {
  function verify(n: number) {
    expect(n).toBe(3);
  }
  it('L85 valid uses verify', () => {
    verify(sum(1, 2));
  });
});
