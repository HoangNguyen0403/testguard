import { expect as chaiExpect } from 'chai';
import * as td from 'testdouble';
import { svc, sum, Emitter, load, names } from './sut';

type Checker = (v: number) => void;

const typedCheck: Checker = (v) => {
  expect(v).toBe(3);
};

class Page {
  expectTitle(t: string) {
    expect(svc.title()).toBe(t);
  }
}

describe('library assertion shapes, no finding expected', () => {
  it('generic expect call', () => {
    expect<number>(sum(1, 2)).toBe(3);
  });

  it('chai getter empty', () => {
    expect(svc.list()).to.be.empty;
  });

  it('sinon-chai calledOnce getter', () => {
    const spy = sinon.spy();
    new Emitter(spy).emit();
    expect(spy).to.have.been.calledOnce;
  });

  it('chai-as-promised rejected', async () => {
    await expect(load('bad')).to.be.rejected;
  });

  it('testdouble verify', () => {
    const save = td.func();
    new Emitter(save).emit();
    td.verify(save(1));
  });

  it('angular http mock verify', () => {
    svc.fetch();
    httpMock.expectOne('/api/items').flush([]);
    httpMock.verify();
  });

  it('typed const helper', () => {
    typedCheck(sum(1, 2));
  });

  it('page object method helper', () => {
    new Page().expectTitle('home');
  });

  it('regex after arrow containing quote', () => {
    const quoted = names.filter((n) => /["']/.test(n));
    expect(quoted).toHaveLength(1);
  });

  it('following test keeps its own body', () => {
    expect(sum(2, 2)).toBe(4);
  });

  it('regex after logical-or containing backtick', () => {
    const ok = svc.ready() || /`/.test(svc.title());
    expect(ok).toBe(true);
  });

  it('last test still discovered', () => {
    expect(sum(3, 3)).toBe(6);
  });

  it('fast-check property assertion', () => {
    fc.assert(fc.property(fc.integer(), (n) => Number.isInteger(n)));
  });
});

function test(a: number, b: number) {
  return a + b;
}

const it2 = { test(x: number) { return x; } };
