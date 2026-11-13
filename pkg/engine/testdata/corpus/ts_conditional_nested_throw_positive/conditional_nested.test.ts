import { parse } from './sut';

it('nested conditional throw is not total', () => {
  const r = parse('x');
  if (r.ok) {
    expect(r.value).toBe('x');
  } else {
    if (r.kind === 'a') {
      throw new Error('unexpected a');
    }
  }
});
