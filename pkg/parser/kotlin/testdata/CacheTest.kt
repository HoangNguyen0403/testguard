package demo

import org.junit.Assert.assertEquals
import org.junit.Test

class CacheTest {
    @Test
    fun fetchReturnsCachedValueSynchronously() {
        val cache = Cache(mapOf(1 to "a"))
        val got = cache.fetch(1)
        assertEquals("a", got)
    }

    @Test
    fun searchIsCaseInsensitive() {
        val index = Index(listOf("Apple"))
        assertEquals(listOf("Apple"), index.search("apple"))
    }
}
