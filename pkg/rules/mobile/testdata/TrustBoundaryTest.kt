import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.StandardTestDispatcher
import org.junit.Test
import kotlin.test.assertEquals

class TrustBoundaryTest {
    private val dispatcher = StandardTestDispatcher()

    @Test
    fun synchronousWorkInsideRunTest() = runTest(dispatcher) {
        val actual = "hello".uppercase()
        assertEquals("HELLO", actual)
    }

    @Test
    fun unrelatedStubDoesNotInvalidateRealResult() {
        every { repo.count() } returns 3
        val actual = Calculator().add(1, 2)
        assertEquals(3, actual.value)
    }

    @Test
    fun commentIsNotAStub() {
        // every { repo.count() } returns 3
        val actual = Calculator().add(1, 2)
        assertEquals(3, actual.value)
    }
}
