import { fit } from './chart-layout';
import { sum, bounds } from './sut';

it('imported fit utility with a callback is not a focused test', () => {
  const scale = fit(bounds, () => 2);
  expect(scale).toBe(2);
});

it('a genuine declaration still analyses normally', () => {
  expect(sum(1, 2)).toBe(3);
});
