import org.junit.Assert.assertEquals
import org.junit.Test

private fun add(a: Int, b: Int): Int = a + b

class VagueTest {
    @Test
    fun foo() {
        assertEquals(3, add(1, 2))
    }
}
