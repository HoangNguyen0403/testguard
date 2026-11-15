package probe

class FUnterminatedRawTest {
    private val broken = """never closed

    @Test
    fun f5() {
        assertEquals(1, 1)
    }
}
