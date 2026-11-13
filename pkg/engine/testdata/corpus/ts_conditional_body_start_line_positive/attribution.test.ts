import { parse } from './sut';

describe('raw body line attribution', () => {
  it(
    'multi-line titled conditional assertion',
    () => {
      const r = parse('x');
      if (r) {
        expect(r.kind).toBe('x');
      }
    }
  );
});
