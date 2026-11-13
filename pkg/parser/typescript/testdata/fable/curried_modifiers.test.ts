import { describe, it, expect, vi } from 'vitest';
import { render } from './render';
import { UserService } from './user.service';
import { sum, sleep } from './sut';

vi.mock('./api');
vi.mock('./user.service', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./user.service')>();
  return { ...actual, audit: vi.fn() };
});

function test_data(n: number) {
  return { n };
}

describe('misc', () => {
  it.skipIf(process.platform === 'win32')('L17 conditional skipIf', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it.runIf(process.env.CI)('L21 conditional runIf', () => {
    expect(sum(1, 2)).toBe(3);
  });

  it.concurrent('L25 concurrent', async () => {
    expect(sum(2, 2)).toBe(4);
  });

  it.fails('L29 expected failure', () => {
    expect(sum(1, 2)).toBe(4);
  });

  it('L33 regex literal mentioning it.skip( and it.only(', () => {
    const re = /it\.skip\(|it\.only\(/;
    expect(re.test('x')).toBe(false);
  });

  it('L38 partial mock keeps real service', () => {
    expect(new UserService().name()).toBe('real');
  });

  it('L42 waits with real timer zero and uses test_data', async () => {
    await sleep(0);
    expect(test_data(1).n).toBe(1);
  });

  it(
    'L48 multiline declaration',
    async () => {
      const v = sum(1, 2);
      expect(v).toBe(v);
    },
  );

  // testguard-disable-next-line GUARD-UNIV-ZERO
  it('L57 suppressed smoke', () => {
    render();
  });

  it('L61 unicode title — naïve café ✓', () => {
    const naïve = sum(1, 2);
    expect(naïve).toBe(3);
  });
});
