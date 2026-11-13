import { sum, cases, parse } from './sut';

const TITLE = 'L3 constant title';

describe('dynamic and unusual declarations (all valid, 0 findings expected)', () => {
  cases.forEach(({ name, input, want }) => {
    it(name, () => {
      expect(sum(input, 0)).toBe(want);
    });
  });

  for (const c of cases) {
    test(c.name, () => {
      expect(sum(c.input, 0)).toBe(c.want);
    });
  }

  it(TITLE, () => {
    expect(sum(1, 1)).toBe(2);
  });

  it('L22 concatenated ' + 'title', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it(`L26 template ${TITLE}`, () => {
    expect(sum(2, 2)).toBe(4);
  });

  it('L30 function expression callback', function () {
    expect(sum(2, 3)).toBe(5);
  });

  it('L34 named function expression callback', function namedCallback() {
    expect(sum(3, 3)).toBe(6);
  });

  it('L38 options object before callback', { timeout: 1000 }, () => {
    expect(sum(3, 4)).toBe(7);
  });

  it('L42 timeout after callback', () => {
    expect(sum(4, 4)).toBe(8);
  }, 10000);

  it('L46 named function used as forEach callback', () => {
    [1, 2].forEach(function checkItem(n) {
      expect(sum(n, 0)).toBe(n);
    });
  });

  it('L52 hoisted local function called before declaration', () => {
    verifyParsed();
    function verifyParsed() {
      expect(parse('x').ok).toBe(true);
    }
  });

  it('L59 promise executor named function', () => {
    return new Promise(function executor(resolve) {
      expect(sum(5, 5)).toBe(10);
      resolve(undefined);
    });
  });

  it.each([[1, 1]])('L66 each %i', (a, b) => {
    expect(sum(a, 0)).toBe(b);
  });

  it.concurrent.each([[2, 2]])('L70 concurrent each %i', async (a, b) => {
    expect(sum(a, 0)).toBe(b);
  });

  specify('L74 mocha specify alias', () => {
    expect(sum(6, 6)).toBe(12);
  });
});

describe.each([[1], [2]])('L79 suite %i', (n) => {
  const check = (v: number) => {
    expect(v).toBe(n);
  };

  describe('nested', () => {
    it('L85 helper from enclosing describe', () => {
      check(sum(n, 0));
    });
  });

  it('L90 arrow helper declared later in same block', () => {
    later(sum(n, 0));
  });

  const later = (v: number) => {
    expect(v).toBe(n);
  };
});
