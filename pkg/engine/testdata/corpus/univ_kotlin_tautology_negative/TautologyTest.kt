class TautologyTest {
    @Test
    fun testValid() {
        assertTrue(service.isReady())
        assertEquals(42, service.getCount())
    }
}
