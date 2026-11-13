import { parse } from './sut';

describe('exhaustive failure narrowing', () => {
  it('else branch fails explicitly', () => {
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      fail('expected ok');
    }
  });

  it('switch default throws', () => {
    const r = parse('x');
    switch (r.kind) {
      case 'a':
        expect(r.value).toBe('a');
        break;
      default:
        throw new Error('unexpected kind');
    }
  });
});
