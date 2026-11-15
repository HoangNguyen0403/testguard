import {it, expect} from 'vitest';
it('unterminated body cannot be complete', () => {
  expect(answer()).toBe(42);
