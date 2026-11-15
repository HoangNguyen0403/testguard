import { it, expect } from 'vitest';
import { sum } from './sut';

describe('curried and parameterised declaration modifiers', () => {
  it.skipIf(process.platform === 'win32')('windows-only skip', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it.runIf(process.env.CI)('ci-only run', () => {
    expect(sum(2, 2)).toBe(4);
  });

  it.concurrent('runs concurrently', async () => {
    expect(sum(3, 3)).toBe(6);
  });

  it.fails('expected failure', () => {
    expect(sum(4, 4)).toBe(9);
  });

  it.each([[1, 2, 3], [2, 3, 5]])('adds %i and %i', (a, b, want) => {
    expect(sum(a, b)).toBe(want);
  });
});
