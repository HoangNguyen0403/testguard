import request from 'supertest';
import assert from 'node:assert';
import sinon from 'sinon';
import fc from 'fast-check';
import { expectTypeOf } from 'vitest';
import { assertValidUser } from './test-helpers';
import { app, parse, svc, load, sum, Emitter, clone } from './sut';

describe('valid tests, no finding expected', () => {
  it('L10 supertest status expectation', async () => {
    await request(app).get('/users').expect(200);
  });

  it('L14 node assert strictEqual', () => {
    assert.strictEqual(sum(1, 2), 3);
  });

  it('L18 bare node assert call', () => {
    assert(svc.ready());
  });

  it('L22 sinon assert', () => {
    const spy = sinon.spy();
    new Emitter(spy).emit();
    sinon.assert.calledOnce(spy);
  });

  it('L28 imported assertion helper', () => {
    assertValidUser(svc.user());
  });

  it('L32 property based', () => {
    fc.assert(fc.property(fc.integer(), (n) => sum(n, 0) === n));
  });

  it('L36 type level expectation', () => {
    expectTypeOf(sum).returns.toBeNumber();
  });

  it('L40 exhaustive result narrowing with throw in else', () => {
    const r = parse('x');
    if (r.ok) {
      expect(r.value).toBe('x');
    } else {
      throw new Error('expected ok');
    }
  });

  it('L49 early return guard for platform', () => {
    if (process.platform === 'win32') {
      return;
    }
    expect(svc.path()).toBe('/tmp/x');
  });

  it('L56 unconditional assertion plus conditional extra', () => {
    const r = parse('x');
    expect(r.ok).toBe(true);
    if (r.ok) {
      expect(r.value).toBe('x');
    }
  });

  it('L64 try with fail call then catch asserts', async () => {
    try {
      await load('bad');
      fail('should have thrown');
    } catch (e) {
      expect((e as Error).message).toMatch(/bad/);
    }
  });

  it('L73 try with expect.fail then catch asserts', async () => {
    try {
      await load('bad');
      expect.fail('should have thrown');
    } catch (e) {
      expect((e as Error).message).toMatch(/bad/);
    }
  });

  it('L82 try with unreachable expect then catch asserts', async () => {
    try {
      await load('bad');
      expect(true).toBe(false);
    } catch (e) {
      expect((e as Error).message).toMatch(/bad/);
    }
  });

  it('L91 mock configured then passed to sut', () => {
    const fetcher = jest.fn().mockResolvedValue({ id: 1 });
    return load('ok', fetcher).then((u) => {
      expect(u.id).toBe(1);
      expect(fetcher).toHaveBeenCalledWith('ok');
    });
  });

  it('L99 clone equals original', () => {
    const original = { a: 1 };
    expect(clone(original)).toEqual(original);
    expect(clone(original)).not.toBe(original);
  });

  it('L105 regex with quote and brace', () => {
    const re = /['"`{(]/;
    expect(re.test("it's")).toBe(true);
  });

  it('L110 template literal with nested braces', () => {
    const s = `a ${[1, 2].map((n) => `${n}}`).join('{')} b`;
    expect(s).toContain('1}');
  });

  it('L115 division is not a regex', () => {
    const a = 10 / 2; const b = 8 / 4;
    expect(a / b).toBe(2.5);
  });

  it('L120 done callback', (done) => {
    new Emitter((v: number) => {
      expect(v).toBe(1);
      done();
    }).emit();
  });

  it('L127 awaited floating then chained', async () => {
    await expect(load('ok')).resolves.toMatchObject({ id: 1 });
  });

  it('L131 resolves inside awaited Promise.allSettled', async () => {
    const results = [expect(load('a')).resolves.toBeTruthy(), expect(load('b')).resolves.toBeTruthy()];
    await Promise.all(results);
  });

  it('L136 toThrow sync', () => {
    expect(() => parse('')).toThrow(/empty/);
  });

  it.todo('L140 pending behaviour');

  // it('commented out', () => {});
  /* it.only('block commented', () => {}); */
});
