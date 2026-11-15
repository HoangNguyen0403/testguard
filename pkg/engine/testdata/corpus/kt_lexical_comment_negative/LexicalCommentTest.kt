class LexicalCommentTest {
    @Test
    fun nestedStringTemplate() {
        val m = mapOf("}" to 1)
        val s = "${m["}"]} end"
        assertEquals("1 end", s)
    }

    /* @Test fun commented() { } /* nested */ @Ignore */
    @Test
    fun afterNestedComment() {
        assertEquals(3, add(1, 2))
    }
}
