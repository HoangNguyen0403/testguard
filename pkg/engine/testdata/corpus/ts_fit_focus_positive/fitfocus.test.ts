import { sum } from './sut';

fit('focused genuine test', () => {
  expect(sum(1, 2)).toBe(3);
});
