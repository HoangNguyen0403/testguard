import { verify } from './helpers';
import { Repository } from './sut';

it('does not credit a verify imported from another module', () => {
  const repository = mock(Repository);
  verify(repository.save('item')).once();
});

it('does not credit a bare verify with no import', () => {
  verify(repository.save('item')).once();
});
