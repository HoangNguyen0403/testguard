import { mock, verify } from 'ts-mockito';
import * as tsm from 'ts-mockito';
import { verify as verifyCalls } from 'ts-mockito';
import { Repository } from './sut';

it('credits a call-terminated verify imported from ts-mockito', () => {
  const repository = mock(Repository);
  repository.save('item');
  verify(repository.save('item')).once();
});

it('credits a pivot-count matcher', () => {
  const repository = mock(Repository);
  verify(repository.save('item')).times(2);
});

it('credits an aliased verify import', () => {
  const repository = mock(Repository);
  verifyCalls(repository.save('item')).never();
});

it('credits a namespace-qualified verify', () => {
  const repository = mock(Repository);
  tsm.verify(repository.save('item')).atLeast(1);
});

it('does not credit a verify call with no matcher', () => {
  const repository = mock(Repository);
  verify(repository.save('item'));
});

it('does not credit a locally shadowed verify', () => {
  const verify = (value: unknown) => value;
  verify(repository.save('item')).once();
});
