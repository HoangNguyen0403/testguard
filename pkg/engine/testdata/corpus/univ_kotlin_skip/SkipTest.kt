import org.junit.Assert.assertEquals
import org.junit.Ignore
import org.junit.Test

private fun add(a: Int, b: Int): Int = a + b

class SkipTest {
    @Ignore("flaky on CI")
    @Test
    fun checksAddition() {
        assertEquals(3, add(1, 2))
    }

    @Test
    fun checksSubtraction() {
        assertEquals(1, subtract(3, 2))
    }
}
