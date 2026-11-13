import { logger } from './sut';

it('nested assertion is credited even when the name appears only in a string', () => {
  function handleClick(e: { type: string }) {
    expect(e.type).toBe('click');
  }
  logger.info('handleClick');
});

it('nested function without a recognized assertion stays hollow', () => {
  function observe(e: { type: string }) {
    logger.info(e.type);
  }
  observe({ type: 'click' });
});
