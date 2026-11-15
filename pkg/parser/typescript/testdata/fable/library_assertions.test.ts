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
  it('L18 generic expect call', () => {
    expect<number>(sum(1, 2)).toBe(3);
  });

  it('L22 chai getter empty', () => {
    expect(svc.list()).to.be.empty;
  });

  it('L26 sinon-chai calledOnce getter', () => {
    const spy = sinon.spy();
    new Emitter(spy).emit();
    expect(spy).to.have.been.calledOnce;
  });

  it('L32 chai-as-promised rejected', async () => {
    await expect(load('bad')).to.be.rejected;
  });

  it('L36 testdouble verify', () => {
    const save = td.func();
    new Emitter(save).emit();
    td.verify(save(1));
  });

  it('L42 angular http mock verify', () => {
    svc.fetch();
    httpMock.expectOne('/api/items').flush([]);
    httpMock.verify();
  });

  it('L48 typed const helper', () => {
    typedCheck(sum(1, 2));
  });

  it('L52 page object method helper', () => {
    new Page().expectTitle('home');
  });

  it('L56 regex after arrow containing quote', () => {
    const quoted = names.filter((n) => /["']/.test(n));
    expect(quoted).toHaveLength(1);
  });

  it('L61 following test keeps its own body', () => {
    expect(sum(2, 2)).toBe(4);
  });

  it('L65 regex after logical-or containing backtick', () => {
    const ok = svc.ready() || /`/.test(svc.title());
    expect(ok).toBe(true);
  });

  it('L70 last test still discovered', () => {
    expect(sum(3, 3)).toBe(6);
  });
});

function test(a: number, b: number) {
  return a + b;
}

const it2 = { test(x: number) { return x; } };
