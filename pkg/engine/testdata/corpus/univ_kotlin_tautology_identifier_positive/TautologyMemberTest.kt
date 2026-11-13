class TautologyMemberTest {
    @Test
    fun sameIdentifier() {
        val r = build()
        assertEquals(r, r)
    }

    @Test
    fun differentReceiversIsNotTautology() {
        val a = build()
        val b = build()
        assertEquals(a.id, b.id)
    }
}
