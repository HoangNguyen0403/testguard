package probe

class DRawBackslashTest {
    private val windowsPath = """C:\Users\dev\"""
    private val regex = """\d+\s"(\w+)"""".toRegex()

    @Test
    fun d8() {
        assertEquals(13, windowsPath.length)
    }

    @Test
    fun d13() {
        assertTrue(regex.containsMatchIn("12 \"ab\""))
    }
}
