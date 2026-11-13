import org.junit.Assert.assertEquals
import org.junit.Test

private fun add(a: Int, b: Int): Int = a + b

class DuplicatesTest {
    @Test
    fun addsTwoNumbers() {
        assertEquals(3, add(1, 2))
    }

    @Test
    fun addsTwoNumbersAgain() {
        assertEquals(3, add(1, 2))
    }
}
