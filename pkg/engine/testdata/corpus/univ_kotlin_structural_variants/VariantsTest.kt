import org.junit.Assert.assertEquals
import org.junit.Test

private fun add(a: Int, b: Int): Int = a + b

class VariantsTest {
    @Test
    fun addsOneAndOne() {
        val input = 1
        val expected = 2
        val result = add(input, input)
        assertEquals(expected, result)
    }

    @Test
    fun addsTwoAndTwo() {
        val input = 2
        val expected = 4
        val result = add(input, input)
        assertEquals(expected, result)
    }

    @Test
    fun addsThreeAndThree() {
        val input = 3
        val expected = 6
        val result = add(input, input)
        assertEquals(expected, result)
    }

    @Test
    fun addsFourAndFour() {
        val input = 4
        val expected = 8
        val result = add(input, input)
        assertEquals(expected, result)
    }
}
