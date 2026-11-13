import assert from 'node:assert';
import request from 'supertest';
import sinon from 'sinon';
import { expectTypeOf } from 'vitest';
import { add, app, stream } from './sut';

describe('assertion styles that are real assertions', () => {
  it('node assert function', () => {
    assert(add(1, 2) === 3);
  });

  it('node assert method', () => {
    assert.strictEqual(add(1, 2), 3);
  });

  it('chai should', () => {
    add(1, 2).should.equal(3);
  });

  it('playwright soft', async () => {
    await expect.soft(page.locator('h1')).toHaveText('Hi');
  });

  it('type test', () => {
    expectTypeOf(add(1, 2)).toBeNumber();
  });

  it('rxjs marble', () => {
    scheduler.run(({ expectObservable }) => {
      expectObservable(stream()).toBe('a|');
    });
  });

  it('supertest status', async () => {
    await request(app).get('/health').expect(200);
  });

  it('sinon assert', () => {
    const spy = sinon.spy();
    add(1, 2);
    sinon.assert.notCalled(spy);
  });

  it('only declares assertion count', () => {
    expect.assertions(1);
    add(1, 2);
  });
});
