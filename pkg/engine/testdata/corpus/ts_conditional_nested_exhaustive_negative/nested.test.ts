import { parse } from './sut';

describe('nested failure narrowing', () => {
  it('nested exhaustive failure branches', () => {
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      if (r.kind === 'a') {
        throw new Error('kind a');
      } else {
        throw new Error('kind b');
      }
    }
  });

  it('nested if with one failing path', () => {
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      if (r.kind === 'a') {
        throw new Error('kind a');
      } else {
        recordMissing(r);
      }
    }
  });
});
