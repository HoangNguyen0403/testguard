package probe

class AFourQuotesTest {
    private val fourQuotes = """ends with quote""""

    @Test
    fun a7() {
        assertEquals(16, fourQuotes.length)
    }
}
