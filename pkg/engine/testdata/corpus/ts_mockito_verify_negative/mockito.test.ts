import { mock, verify } from 'ts-mockito';
import { Repository, Service } from './sut';

it('delegates saving through the repository', () => {
  const repository = mock(Repository);
  const service = new Service(repository);
  service.save('item');
  verify(repository.save('item')).once();
});
