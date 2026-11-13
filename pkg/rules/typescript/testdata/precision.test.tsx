import { render, screen } from '@testing-library/react';
import { a, b, f, parse, Panel, emitter, svc } from './sut';

function expectOk(r: { ok: boolean }) {
  expect(r.ok).toBe(true);
}

describe('correct tests that must produce no finding', () => {
  it('awaits all async assertions in an array', async () => {
    await Promise.all([expect(a()).resolves.toBe(1), expect(b()).rejects.toThrow()]);
  });

  it('returns the async assertion', () => {
    return expect(a()).resolves.toBe(1);
  });

  it('expression-bodied async assertion', () => expect(a()).resolves.toBe(1));

  it('stores then awaits the assertion', async () => {
    const pending = expect(b()).rejects.toThrow();
    await pending;
  });

  it('narrows then asserts unconditionally', () => {
    const r = parse('x');
    if (!r) throw new Error('unreachable');
    expect(r.kind).toBe('x');
  });

  it('asserts inside a loop', () => {
    for (const n of [1, 2, 3]) {
      expect(f(n)).toBeGreaterThan(0);
    }
  });

  it('error path with assertion count', async () => {
    expect.assertions(1);
    try {
      await b();
    } catch (e) {
      expect(e).toBeInstanceOf(Error);
    }
  });

  it('error path with explicit fail', async () => {
    try {
      await b();
      throw new Error('should have thrown');
    } catch (e) {
      expect((e as Error).message).toBe('boom');
    }
  });

  it('callback mock invoked by the emitter', () => {
    const cb = jest.fn();
    emitter.on('x', cb);
    emitter.emit('x');
    expect(cb).toHaveBeenCalledTimes(1);
  });

  it('callback mock invoked by forEach', () => {
    const cb = jest.fn();
    [1, 2].forEach(cb);
    expect(cb).toHaveBeenCalledTimes(2);
  });

  it('presence of rendered text', () => {
    render(<Panel title="it's here" />);
    expect(screen.getByText("it's here")).toBeInTheDocument();
  });

  it('jsx text with an apostrophe before the next test', () => {
    render(<p>don't break the lexer {'}'}</p>);
    expect(screen.getByText(/don't/)).toBeVisible();
  });

  it('defined plus a value check', () => {
    expect(svc).toBeDefined();
    expect(svc.count()).toBe(2);
  });

  it('uses a same-file asserting helper', () => {
    expectOk(svc.status());
  });

  it('chai property assertion', () => {
    expect(svc.ready).to.be.true;
  });

  it('regex method named test is not a test declaration', () => {
    const matched = /it\('x'\)/.test("it('x')");
    expect(matched).toBe(true);
  });

  it('string that looks like a test', () => {
    const source = "it('fake', () => {})";
    expect(source.length).toBe(20);
  });
});

describe.each([[1], [2]])('parameterised suite %i', (n) => {
  it('inner test', () => {
    expect(f(n)).toBe(n);
  });
});

test.each`
  x    | y
  ${1} | ${1}
`('tagged table $x', ({ x, y }) => {
  expect(f(x)).toBe(y);
});
