import org.junit.Assert.assertEquals
import org.junit.Test

private fun add(a: Int, b: Int): Int = a + b

class VariantsTest {
    @Test
    fun addsOneAndOne() {
        assertEquals(2, add(1, 1))
    }

    @Test
    fun addsTwoAndTwo() {
        assertEquals(4, add(2, 2))
    }

    @Test
    fun addsThreeAndThree() {
        assertEquals(6, add(3, 3))
    }

    @Test
    fun addsFourAndFour() {
        assertEquals(8, add(4, 4))
    }
}
