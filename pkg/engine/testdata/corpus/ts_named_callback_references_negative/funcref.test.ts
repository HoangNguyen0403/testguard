import { Emitter, sum } from './sut';

describe('named functions that run without being called by name (valid, 0 findings expected)', () => {
  it('L4 handler declared then passed by reference', (done) => {
    function onData(v: number) {
      expect(v).toBe(1);
      done();
    }
    const emitter = new Emitter();
    emitter.on('data', onData);
    emitter.emit('data', 1);
  });

  it('L14 checker declared then passed to forEach', () => {
    function checkItem(n: number) {
      expect(sum(n, 0)).toBe(n);
    }
    [1, 2, 3].forEach(checkItem);
  });

  it('L21 function assigned then invoked through alias', () => {
    function verifyTotal() {
      expect(sum(2, 2)).toBe(4);
    }
    const run = verifyTotal;
    run();
  });

  it('L29 mocha style named callback', function mochaStyle() {
    expect(sum(1, 2)).toBe(3);
  });
});

describe('still hollow (ZERO expected)', () => {
  it('L35 declared and never referenced', () => {
    function unused() {
      expect(sum(1, 2)).toBe(3);
    }
    sum(1, 2);
  });
});
